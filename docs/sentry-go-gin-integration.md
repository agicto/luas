# Sentry Go SDK + Gin 集成指南

> 深入理解 Sentry Go SDK 运作逻辑，针对 Gin 框架的完整集成方案

## 目录

1. [Sentry Go SDK 核心架构](#1-sentry-go-sdk-核心架构)
2. [Gin 中间件集成](#2-gin-中间件集成)
3. [完整集成示例](#3-完整集成示例)
4. [高级功能](#4-高级功能)
5. [Trac 服务器兼容性](#5-trac-服务器兼容性)

---

## 1. Sentry Go SDK 核心架构

### 1.1 核心概念

```
┌─────────────────────────────────────────────────────────────┐
│                     Sentry Go SDK                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌─────────┐    ┌─────────┐    ┌───────────┐               │
│   │  Hub    │───▶│  Scope  │───▶│  Client   │               │
│   └─────────┘    └─────────┘    └───────────┘               │
│        │              │              │                       │
│        │              │              ▼                       │
│        │              │        ┌───────────┐                │
│        │              │        │ Transport │                │
│        │              │        └───────────┘                │
│        │              │              │                       │
│        ▼              ▼              ▼                       │
│   ┌─────────────────────────────────────┐                   │
│   │         Event / Transaction         │                   │
│   └─────────────────────────────────────┘                   │
│                      │                                       │
│                      ▼                                       │
│   ┌─────────────────────────────────────┐                   │
│   │    POST /api/:project_id/envelope/  │ ──▶ Trac Server   │
│   └─────────────────────────────────────┘                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Hub（中心枢纽）

Hub 是 Sentry SDK 的核心管理器，负责：
- 管理 Scope 栈（支持嵌套上下文）
- 持有 Client 引用
- 提供事件捕获入口

```go
// 获取当前 Hub
hub := sentry.CurrentHub()

// 克隆 Hub（用于并发场景）
clonedHub := hub.Clone()

// 获取最后一个事件 ID
eventID := hub.LastEventID()
```

### 1.3 Scope（作用域）

Scope 存储上下文数据，会附加到所有捕获的事件上：

```go
sentry.ConfigureSa scope := hub.Scope()

// 设置用户
scope.SetUser(sentry.User{
    ID:       "user-123",
    Email:    "user@example.com",
    Username: "john_doe",
})

// 设置标签
scope.SetTag("environment", "production")
scope.SetTag("server", "web-01")

// 设置额外数据
scope.SetExtra("request_id", "abc-123")

// 设置上下文
scope.SetContext("browser", map[string]interface{}{
    "name":    "Chrome",
    "version": "120.0",
})

// 设置事件级别
scope.SetLevel(sentry.LevelError)

// 设置 Fingerprint（控制问题分组）
scope.SetFingerprint([]string{"database", "connection", "timeout"})
```

### 1.4 Client（客户端）

Client 负责：
- 配置管理（DSN、采样率等）
- 事件处理管道（BeforeSend、EventProcessor）
- 与 Transport 交互

```go
// 初始化 Client
err := sentry.Init(sentry.ClientOptions{
    Dsn:              "http://key@your-trac-server:8025/1",
    Debug:            true,
    AttachStacktrace: true,
    SampleRate:       1.0,           // 100% 采样
    TracesSampleRate: 0.2,           // 20% 性能追踪采样
    Environment:      "production",
    Release:          "myapp@1.0.0",
    ServerName:       "web-server-01",
    MaxBreadcrumbs:   100,
    
    // 事件过滤
    BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
        // 可以修改或丢弃事件
        if event.Level == sentry.LevelDebug {
            return nil // 丢弃 debug 级别事件
        }
        return event
    },
})
```

### 1.5 Transport（传输层）

Transport 负责将事件发送到 Sentry/Trac 服务器：

```go
// 默认使用 HTTPTransport（异步、非阻塞）
// - BufferSize: 1000（内存队列大小）
// - Timeout: 30秒

// 自定义 Transport
transport := sentry.NewHTTPTransport()
transport.BufferSize = 100
transport.Timeout = 10 * time.Second

sentry.Init(sentry.ClientOptions{
    Transport: transport,
})
```

### 1.6 Envelope 协议

SDK 使用 Envelope 格式发送数据（换行分隔的 JSON）：

```
{"event_id":"...","sent_at":"2026-01-06T15:00:00Z","dsn":"...","sdk":{"name":"sentry.go","version":"0.40.0"}}
{"type":"event","length":1234}
{"level":"error","message":"Something went wrong","exception":[...],"breadcrumbs":[...]}
```

---

## 2. Gin 中间件集成

### 2.1 sentrygin 中间件工作原理

```go
// sentrygin.New() 返回的中间件执行流程：
func (h *handler) handle(c *gin.Context) {
    // 1. 获取或克隆 Hub
    hub := sentry.GetHubFromContext(ctx)
    if hub == nil {
        hub = sentry.CurrentHub().Clone()
    }
    
    // 2. 启动 Transaction（性能追踪）
    transaction := sentry.StartTransaction(
        sentry.SetHubOnContext(ctx, hub),
        fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()),
        // 继续分布式追踪
        sentry.ContinueTrace(hub, c.GetHeader("sentry-trace"), c.GetHeader("baggage")),
        sentry.WithOpName("http.server"),
    )
    
    // 3. 设置请求上下文
    hub.Scope().SetRequest(c.Request)
    c.Set("sentry", hub)
    c.Set("sentry_transaction", transaction)
    
    // 4. Panic Recovery
    defer func() {
        if err := recover(); err != nil {
            hub.RecoverWithContext(ctx, err)
            if h.waitForDelivery {
                hub.Flush(h.timeout)
            }
            if h.repanic {
                panic(err) // 重新抛出
            }
        }
    }()
    
    // 5. 继续处理请求
    c.Next()
    
    // 6. 结束 Transaction
    transaction.Status = sentry.HTTPtoSpanStatus(c.Writer.Status())
    transaction.Finish()
}
```

### 2.2 关键特性

| 特性 | 说明 |
|------|------|
| **Panic Recovery** | 自动捕获 panic 并上报 |
| **Transaction 追踪** | 记录请求耗时和状态 |
| **分布式追踪** | 支持 `sentry-trace` 和 `baggage` 头 |
| **Hub 隔离** | 每个请求独立的 Hub，避免并发问题 |

---

## 3. 完整集成示例

### 3.1 基础集成

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/getsentry/sentry-go"
    sentrygin "github.com/getsentry/sentry-go/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    // 初始化 Sentry
    if err := sentry.Init(sentry.ClientOptions{
        // 指向 Trac 服务器
        Dsn: "http://your_public_key@localhost:8025/1",
        
        // 开启调试模式（生产环境关闭）
        Debug: true,
        
        // 自动附加堆栈
        AttachStacktrace: true,
        
        // 采样率
        SampleRate:       1.0,  // 错误 100% 采样
        TracesSampleRate: 0.2,  // 性能 20% 采样
        
        // 环境和版本
        Environment: "development",
        Release:     "myapp@1.0.0",
        
        // 事件处理
        BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
            // 过滤敏感信息
            if event.Request != nil {
                delete(event.Request.Headers, "Authorization")
            }
            return event
        },
    }); err != nil {
        log.Fatalf("sentry.Init: %v", err)
    }
    defer sentry.Flush(2 * time.Second)

    // 创建 Gin 引擎
    r := gin.Default()
    
    // 添加 Sentry 中间件
    r.Use(sentrygin.New(sentrygin.Options{
        Repanic:         true,  // 重新抛出 panic（让 Gin Recovery 处理响应）
        WaitForDelivery: false, // 不阻塞请求
    }))

    // 路由
    r.GET("/", func(c *gin.Context) {
        c.String(http.StatusOK, "Hello, World!")
    })

    r.GET("/error", func(c *gin.Context) {
        // 手动捕获错误
        if hub := sentrygin.GetHubFromContext(c); hub != nil {
            hub.CaptureMessage("This is a test error")
        }
        c.String(http.StatusOK, "Error sent to Sentry")
    })

    r.GET("/panic", func(c *gin.Context) {
        // 触发 panic（会被自动捕获）
        panic("Something went terribly wrong!")
    })

    r.Run(":8080")
}
```

### 3.2 高级用法：手动管理 Scope

```go
r.POST("/order", func(c *gin.Context) {
    hub := sentrygin.GetHubFromContext(c)
    if hub == nil {
        return
    }
    
    // 使用 WithScope 添加请求级别上下文
    hub.WithScope(func(scope *sentry.Scope) {
        // 设置用户
        scope.SetUser(sentry.User{
            ID:    c.GetHeader("X-User-ID"),
            Email: c.GetHeader("X-User-Email"),
        })
        
        // 设置标签
        scope.SetTag("order_type", "premium")
        
        // 设置上下文
        scope.SetContext("order", map[string]interface{}{
            "order_id": c.Param("id"),
            "amount":   1999.99,
        })
        
        // 添加面包屑
        hub.AddBreadcrumb(&sentry.Breadcrumb{
            Category: "order",
            Message:  "Processing order",
            Level:    sentry.LevelInfo,
        }, nil)
        
        // 业务逻辑
        if err := processOrder(c); err != nil {
            hub.CaptureException(err)
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
    })
    
    c.JSON(200, gin.H{"status": "success"})
})
```

### 3.3 自定义 Fingerprint

```go
r.GET("/api/users/:id", func(c *gin.Context) {
    hub := sentrygin.GetHubFromContext(c)
    
    hub.WithScope(func(scope *sentry.Scope) {
        // 自定义问题分组
        scope.SetFingerprint([]string{
            "user-api",
            "get-user",
            c.Request.URL.Path,
        })
        
        if err := getUserByID(c.Param("id")); err != nil {
            hub.CaptureException(err)
        }
    })
})
```

---

## 4. 高级功能

### 4.1 Breadcrumbs（面包屑）

记录导致错误的事件链：

```go
// 手动添加
sentry.AddBreadcrumb(&sentry.Breadcrumb{
    Category:  "auth",
    Message:   "User logged in",
    Level:     sentry.LevelInfo,
    Data: map[string]interface{}{
        "user_id": "123",
    },
})

// HTTP 请求自动记录
// Database 查询需要手动记录
sentry.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "db.query",
    Message:  "SELECT * FROM users WHERE id = ?",
    Level:    sentry.LevelInfo,
    Data: map[string]interface{}{
        "duration_ms": 15,
    },
})
```

### 4.2 Transaction / Span（性能追踪）

```go
r.GET("/checkout", func(c *gin.Context) {
    // 获取当前 Transaction
    span := sentrygin.GetSpanFromContext(c)
    
    // 创建子 Span
    validateSpan := span.StartChild("validation")
    validateSpan.Description = "Validate cart items"
    // ... 验证逻辑
    validateSpan.Finish()
    
    // 另一个子 Span
    paymentSpan := span.StartChild("payment.process")
    paymentSpan.Description = "Process payment with Stripe"
    paymentSpan.SetData("provider", "stripe")
    // ... 支付逻辑
    paymentSpan.Finish()
    
    c.JSON(200, gin.H{"status": "completed"})
})
```

### 4.3 错误过滤

```go
sentry.Init(sentry.ClientOptions{
    // 忽略特定错误
    IgnoreErrors: []string{
        "context canceled",
        "connection reset by peer",
    },
    
    // 忽略特定 Transaction
    IgnoreTransactions: []string{
        "/health",
        "/metrics",
    },
    
    // 自定义过滤
    BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
        // 忽略 404 错误
        if event.Request != nil && event.Tags["http.status_code"] == "404" {
            return nil
        }
        return event
    },
})
```

---

## 5. Trac 服务器兼容性

### 5.1 完全兼容

Trac 服务器实现了 Sentry Envelope 协议，支持：

| 功能 | Sentry | Trac |
|------|--------|------|
| 错误捕获 | ✅ | ✅ |
| 堆栈追踪 | ✅ | ✅ |
| Breadcrumbs | ✅ | ✅ |
| Tags / Extra | ✅ | ✅ |
| User 信息 | ✅ | ✅ |
| Fingerprint 分组 | ✅ | ✅ |
| Issue 状态管理 | ✅ | ✅ |
| Transaction* | ✅ | 🔜 计划中 |

### 5.2 DSN 配置

```go
// Sentry 官方
Dsn: "https://key@o123.ingest.sentry.io/456"

// Trac 服务器（只需替换地址）
Dsn: "http://your_public_key@your-trac-server:8025/1"
```

### 5.3 获取 DSN

```bash
# 通过 API 获取项目 DSN
curl http://localhost:8025/v1/projects/1/dsn \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应
{
  "data": {
    "dsn": "http://abc123@localhost:8025/1"
  }
}
```

---

## 总结

Sentry Go SDK 的核心工作流程：

1. **初始化**：`sentry.Init()` 创建 Client 和全局 Hub
2. **中间件**：`sentrygin` 为每个请求克隆 Hub，启动 Transaction
3. **捕获**：手动 `CaptureException` 或自动 `recover()` panic
4. **上下文**：通过 Scope 添加 User、Tags、Breadcrumbs
5. **传输**：HTTPTransport 异步发送 Envelope 到服务器
6. **聚合**：Trac 根据 Fingerprint 将事件聚合为 Issue

只需将 DSN 指向 Trac 服务器，即可无缝迁移！
