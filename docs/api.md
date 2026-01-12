# Trac API 接口文档

> 轻量化 Sentry 替代方案 - 完全兼容 Sentry Go/JS/Python SDK

## 快速开始

### Sentry SDK 兼容性

**是的，用户可以直接使用官方 Sentry SDK 上报错误到 Trac 系统！** 只需修改 DSN 地址即可，无需任何代码改动。

```go
// Go SDK 示例
import "github.com/getsentry/sentry-go"

func main() {
    sentry.Init(sentry.ClientOptions{
        // 只需将 DSN 改为 Trac 服务器地址
        Dsn: "http://{PUBLIC_KEY}@your-trac-server.com:8025/{PROJECT_ID}",
    })
    defer sentry.Flush(2 * time.Second)
    
    sentry.CaptureMessage("Hello from Sentry SDK!")
}
```

```javascript
// JavaScript SDK 示例
import * as Sentry from "@sentry/browser";

Sentry.init({
  dsn: "http://{PUBLIC_KEY}@your-trac-server.com:8025/{PROJECT_ID}",
});
```

```python
# Python SDK 示例
import sentry_sdk

sentry_sdk.init(
    dsn="http://{PUBLIC_KEY}@your-trac-server.com:8025/{PROJECT_ID}",
)
```

---

## DSN 格式

```
{PROTOCOL}://{PUBLIC_KEY}@{HOST}/{PROJECT_ID}

示例: http://abc123def456@localhost:8025/1
```

| 字段 | 说明 |
|------|------|
| `PROTOCOL` | `http` 或 `https` |
| `PUBLIC_KEY` | 项目公钥（通过 `/v1/projects/:id/dsn` 获取） |
| `HOST` | Trac 服务器地址 |
| `PROJECT_ID` | 项目 ID |

---

## API 端点概览

### SDK 上报端点（公开，无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/{project_id}/envelope/` | Sentry Envelope 上报（推荐） |
| `POST` | `/api/{project_id}/store/` | 传统事件上报（已废弃） |

### 项目管理（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/projects` | 创建项目 |
| `GET` | `/v1/projects` | 获取项目列表 |
| `GET` | `/v1/projects/:id` | 获取项目详情 |
| `PUT` | `/v1/projects/:id` | 更新项目 |
| `DELETE` | `/v1/projects/:id` | 删除项目 |
| `GET` | `/v1/projects/:id/dsn` | 获取项目 DSN |

### Issue 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/projects/:id/issues/` | 获取 Issue 列表 |
| `GET` | `/v1/projects/:id/issues/:fingerprint` | 获取 Issue 详情 |
| `POST` | `/v1/projects/:id/issues/:fingerprint/resolve` | 标记为已解决 |
| `POST` | `/v1/projects/:id/issues/:fingerprint/ignore` | 标记为忽略 |
| `POST` | `/v1/projects/:id/issues/:fingerprint/reopen` | 重新打开 |
| `GET` | `/v1/projects/:id/issues/:fingerprint/events` | 获取事件列表 |

---

## 详细 API 说明

### 1. SDK 事件上报

#### POST `/api/{project_id}/envelope/`

Sentry SDK 自动调用此端点上报事件。

**Headers:**
```
Content-Type: application/x-sentry-envelope
X-Sentry-Auth: Sentry sentry_version=7, sentry_client=sentry.go/0.x.x, sentry_key={PUBLIC_KEY}
```

**Envelope 格式（换行分隔）:**
```
{"event_id":"...","sent_at":"...","dsn":"..."}
{"type":"event","length":1234}
{...event JSON payload...}
```

**响应:**
```json
{"id": "event-uuid-here"}
```

---

### 2. 创建项目

#### POST `/v1/projects`

**请求体:**
```json
{
  "name": "My App",
  "platform": "go"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "My App",
    "platform": "go",
    "public_key": "abc123def456"
  }
}
```

---

### 3. 获取 DSN

#### GET `/v1/projects/:id/dsn`

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "dsn": "http://abc123def456@localhost:8025/1"
  }
}
```

---

### 4. 获取 Issue 列表

#### GET `/v1/projects/:id/issues/`

**查询参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码 |
| `per_page` | int | 20 | 每页数量 |
| `status` | string | - | 过滤状态: `unresolved`, `resolved`, `ignored` |
| `level` | string | - | 过滤级别: `debug`, `info`, `warning`, `error`, `fatal` |

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "issues": [
      {
        "ProjectID": 1,
        "Fingerprint": "null-pointer",
        "FirstSeen": "2026-01-05T22:44:58Z",
        "LastSeen": "2026-01-05T22:53:41Z",
        "EventCount": 14,
        "LastLevel": "fatal",
        "Status": "unresolved"
      }
    ],
    "total": 4,
    "page": 1,
    "per_page": 20
  }
}
```

---

### 5. 获取 Issue 详情

#### GET `/v1/projects/:id/issues/:fingerprint`

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ProjectID": 1,
    "Fingerprint": "null-pointer",
    "EventCount": 14,
    "Status": "unresolved",
    "recent_events": [
      {
        "event_id": "abc123",
        "timestamp": "2026-01-05T22:53:41Z",
        "level": "fatal",
        "exception": [
          {
            "type": "*errors.errorString",
            "value": "null pointer dereference",
            "stacktrace": {...}
          }
        ]
      }
    ]
  }
}
```

---

### 6. 管理 Issue 状态

#### POST `/v1/projects/:id/issues/:fingerprint/resolve`
#### POST `/v1/projects/:id/issues/:fingerprint/ignore`
#### POST `/v1/projects/:id/issues/:fingerprint/reopen`

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "fingerprint": "null-pointer",
    "message": "issue resolved"
  }
}
```

---

### 7. 获取 Issue 事件列表

#### GET `/v1/projects/:id/issues/:fingerprint/events`

**查询参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码 |
| `per_page` | int | 10 | 每页数量 |

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "events": [...],
    "total": 14,
    "page": 1,
    "per_page": 10
  }
}
```

---

## 错误码

| 代码 | 说明 |
|------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（缺少或无效的认证） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 支持的 SDK

| SDK | 版本 | 状态 |
|-----|------|------|
| sentry-go | 0.x+ | ✅ 完全支持 |
| @sentry/browser | 7.x+ | ✅ 完全支持 |
| @sentry/node | 7.x+ | ✅ 完全支持 |
| sentry-python | 1.x+ | ✅ 完全支持 |
| sentry-java | 6.x+ | ✅ 完全支持 |

---

## 部署配置

### 环境变量

```bash
# ClickHouse 配置
LOG_CH_ENABLED=true
LOG_CH_ENDPOINT=localhost:9000
LOG_CH_DATABASE=trac
LOG_CH_USERNAME=trac_user
LOG_CH_PASSWORD=trac_pass

# 服务配置
SERVER_PORT=8025
```

### Docker Compose

```yaml
services:
  trac-api:
    image: trac-api:latest
    ports:
      - "8025:8025"
    environment:
      - LOG_CH_ENDPOINT=clickhouse:9000
    depends_on:
      - clickhouse

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    ports:
      - "9000:9000"
```
