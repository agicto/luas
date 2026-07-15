# Luas

> 面向模块化 Go API 的纯脚手架

Luas 是一个用于搭建 Go 后端项目的脚手架，目标是提供稳定的项目结构、依赖注入、模块边界、统一响应、分页、迁移、测试工具和常用基础设施集成。

这个仓库的定位是“框架与模板”，不是某个具体业务系统。默认只保留最小可用的认证、API key 和审计 starter，增强型业务模块和示例能力不再自动挂载到主应用。

## 核心能力

- 模块化目录结构，适合 DDD + 分层架构
- Gin HTTP 服务入口与统一路由注册
- Wire 依赖注入
- GORM 数据访问与迁移体系
- 内置 starter：`user`, `apikey`, `audit`
- Provider-neutral AI capability 与内置 CLI `ai:chat`
- 可取消、可安全关闭的 workflow queue capability 与 worker CLI
- 统一 API 响应与错误处理
- 分页、验证、日志、JWT、中间件
- 测试辅助工具与集成测试基线
- 可选集成：Redis、邮件、OpenTelemetry、R2、Sentry

## 快速开始

### 1. 环境准备

- Go 1.25.12+
- PostgreSQL 12+ 或 SQLite
- Redis 6+（可选）

### 2. 初始化配置

```bash
cp .env.example .env
```

最少需要确认以下配置：

```bash
APP_NAME=Luas
APP_ENV=development
SERVER_HOST=127.0.0.1
SERVER_PORT=8025

DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_NAME=luas

JWT_SECRET=replace-me
```

`user`、`apikey`、`audit` 默认启用。组织与权限是可选 Starter；需要权限时给 HTTP
进程、迁移任务和 seeder 任务统一设置完整依赖：

```bash
OPTIONAL_STARTERS=organization,permission
```

组织 starter 已包含邀请、成员、所有权转移和活跃上下文闭环；权限 starter 提供组织范围的
访问角色、精确授权、事务化分配和可替换 authorizer。见
[`../contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md) 与
[`../contracts/PERMISSIONS.md`](../contracts/PERMISSIONS.md)。

### 3. 生成依赖注入代码

```bash
make wire
```

### 4. 启动 HTTP 服务

```bash
go run ./cmd/server
```

默认监听地址是仅本机可访问的 `127.0.0.1:8025`：

- 应用首页：`http://127.0.0.1:8025/`
- 健康检查：`http://127.0.0.1:8025/v1/health`
- Readiness：`http://127.0.0.1:8025/health/ready`
- Prometheus metrics：开发环境默认 `http://127.0.0.1:8025/metrics`；生产环境需显式设置 `METRICS_ENABLED=true` 并通过网络策略限制访问

### 5. 使用 CLI

```bash
go run ./cmd/luas version
DB_ENABLED=false JWT_SECRET=route-list-inspection-only-0000000000000000 \
  go run ./cmd/luas route:list
go run ./cmd/luas migrate
go run ./cmd/luas seed
go run ./cmd/luas ai:chat "Summarize this scaffold in one sentence"
```

这里内联的 route-list secret 仅用于非服务态检查，禁止在实际运行的 API 中复用。

## 常用命令

```bash
make build
make test
make test-race-critical
make benchmark-http
make benchmark-workflow
make container-check
make compose-check
make lint
make wire
make vuln
make air
```

## 默认 HTTP 防护

API HTTP kernel 默认启用以下 core guardrails：

- `ListenAddress`：默认只绑定 `127.0.0.1`；容器镜像显式设置 `SERVER_HOST=0.0.0.0`
- `TransportTimeouts`：header 读取 10 秒、完整请求读取 60 秒、响应写入 190 秒、keep-alive idle 120 秒
- `HeaderLimit`：默认最多读取 64 KiB request headers

- `RequestID`：为响应和错误输出提供 `X-Request-ID` / `request_id`
- `Helmet`：发送基础安全响应头
- `BodyLimit`：默认 10MB，请求过大返回 `413` + `COMMON.REQUEST_TOO_LARGE`
- `Timeout`：默认 180 秒 cooperative request timeout；handler 尊重 `context` 且未写响应时返回 `503` + `COMMON.TIMEOUT`
- `RateLimit`：`APP_ENV=production` 时默认启用，每个 client IP 默认 `600/min`，超限返回 `429` + `COMMON.RATE_LIMITED`
- `AuthAbuseGuard`：生产环境默认启用；登录和密码重置同时使用独立的 per-IP 与 per-subject 配额
- `TrustedProxies`：默认不信任转发头，只有 `SERVER_TRUSTED_PROXIES` 明确列出的上游才能提供 client IP
- `CORS`：默认只允许本地 Web shell，生产环境必须显式配置可信 origin

可通过 `.env` 调整：

```bash
SERVER_HOST=127.0.0.1
SERVER_READ_TIMEOUT=60
SERVER_READ_HEADER_TIMEOUT=10
SERVER_WRITE_TIMEOUT=190
SERVER_IDLE_TIMEOUT=120
SERVER_MAX_HEADER_BYTES=65536
MIDDLEWARE_REQUEST_TIMEOUT=180
MIDDLEWARE_BODY_LIMIT_MB=10
MIDDLEWARE_RATE_LIMIT_ENABLED=true
MIDDLEWARE_RATE_LIMIT_MAX=600
MIDDLEWARE_RATE_LIMIT_WINDOW=1m
AUTH_RATE_LIMIT_ENABLED=true
AUTH_RATE_LIMIT_LOGIN_IP_MAX=20
AUTH_RATE_LIMIT_LOGIN_SUBJECT_MAX=10
SERVER_TRUSTED_PROXIES=10.20.0.0/16
CORS_ALLOW_ORIGINS=https://app.example.com
```

`SERVER_HOST` 是真实 socket bind 地址，不只是 banner 文本。`SERVER_WRITE_TIMEOUT` 必须大于
`MIDDLEWARE_REQUEST_TIMEOUT`，确保 cooperative timeout 有机会写出标准错误响应；只有明确由网关或
流式端点拥有写入期限时才应设为 `0`。负数 transport 预算和矛盾的超时关系会在启动时失败。

Timeout 不会在 goroutine 中抢占 Gin handler；它通过 request context deadline 让数据库、HTTP client、AI provider 等下游调用安全取消。全局与认证限流都使用进程内 memory store，适合作为 scaffold 的单实例安全默认；多实例生产环境应在网关、WAF、Redis store 或部署层补充分布式限流。认证限流不会返回桶类型或剩余额度，且不能替代 MFA、风险识别和渐进式挑战。Compression 保留给部署/CDN 层或显式 middleware，不在默认 kernel 中重复压缩响应。

完整 middleware 分类见 [docs/MIDDLEWARE.md](docs/MIDDLEWARE.md)。

## 数据库禁用模式

`DB_ENABLED=false` 允许 API 在不创建数据库连接时启动，便于检查 root、health、metrics、
路由装配和下游抽取结果。默认 starter 路由仍然注册；认证、参数约束和输入校验可以先返回
各自的错误，但任何真正触达持久化的操作都会返回 `503` +
`COMMON.SERVICE_UNAVAILABLE`，不会因为 nil GORM 连接 panic。audit 写入保持 best-effort，
失败只记录告警，不得覆盖已经生成的主响应。

该模式表示依赖降级，不表示默认 starter 可用。`/health/live` 继续存活，
`/health/ready` 会因 database 状态为 down 而返回 `503`。保留默认 starter 的生产部署应启用
并连接数据库；只有已经删除所有 DB-backed starter 的下游应用才应把无数据库运行视为完整模式。

Workflow 的 `sync` / `memory` 驱动定位、payload 所有权、关闭语义和生产替换边界见
[docs/WORKFLOW.md](docs/WORKFLOW.md)。`memory` 驱动是有界、进程内、非持久队列，不支持多副本之间的任务传递。

## 项目结构

```text
luas/api/
├── cmd/
│   ├── server/               # HTTP 服务入口
│   └── luas/                  # CLI 入口
├── internal/
│   ├── app/                  # 应用聚合对象
│   ├── bootstrap/            # 启动与生命周期
│   ├── domain/               # 领域对象与领域错误
│   ├── infra/                # 通用基础设施
│   ├── modules/              # 业务模块
│   └── wiring/               # Wire DI
├── pkg/                      # 通用公共包
├── routes/                   # 全局路由入口
├── database/
│   ├── migrations/           # 数据迁移
│   └── seeders/              # 数据初始化
└── tests/
    ├── feature/
    ├── integration/
    └── unit/
```

## 模块约定

默认模块边界：

- `internal/modules/user` 是默认认证 starter，会参与默认路由、迁移和数据初始化
- `internal/modules/apikey` 是默认 API key starter，会参与默认路由和迁移，并提供 `api_key` 中间件组
- `internal/modules/audit` 是默认审计 starter，会记录全局写请求，并提供当前用户的审计历史查询

业务模块建议遵循 8 文件结构：

```text
internal/modules/<module>/
├── model.go
├── dto.go
├── repository.go
├── service.go
├── handler.go
├── routes.go
├── provider.go
└── service_test.go
```

分层流向：

```text
Handler -> Service -> Repository -> Database
DTO -> Domain -> PO
```

约束建议：

- `handler` 负责参数绑定、鉴权上下文和响应输出
- `service` 负责业务规则和错误语义
- `repository` 负责 PO 与 domain 的边界转换
- API 统一走 `pkg/response`
- 列表接口统一使用分页

## 配置生命周期

`internal/infra/config.Config` 是 API 唯一的强类型配置权威。进程启动时按“系统环境变量、
`LUAS_ENV_FILE`、环境本地文件、本地文件、环境文件、基础 `.env`、代码默认值”的顺序
生成并校验一次配置快照；配置变化需要重启进程，开发时 `make air` 会完成重建和重启。

Luas 不提供会泄露密钥的配置缓存，也不提供无法原子重建依赖图的 `.env` 运行时热更新。
完整优先级、扩展规范和 `doctor` 诊断方式见 [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)。

## 测试

```bash
make test
make test-kest
go test ./...
go test ./tests/feature/...
go test ./tests/integration/...
```

Kest flow 入口：

- `tests/kest/auth.flow.md`
- `tests/kest/api_keys.flow.md`

本地一键运行：

```bash
make test-kest
./tests/kest/run_local.sh tests/kest/auth.flow.md
```

## AI Capability

脚手架内置了 provider-neutral 的 `internal/capabilities/ai` 能力层，当前默认接了 OpenAI Responses API。

最小配置：

```bash
AI_ENABLED=true
AI_DEFAULT_PROVIDER=openai
AI_DEFAULT_MODEL=gpt-5
OPENAI_API_KEY=replace-me
```

命令示例：

```bash
go run ./cmd/luas ai:chat "Write a short project summary"
go run ./cmd/luas ai:chat --system="Answer in JSON" --model=gpt-5 "List 3 scaffold priorities"
```

## API Key Starter

脚手架默认内置 API key 管理模块，提供：

- `GET /v1/api-keys`
- `POST /v1/api-keys`
- `DELETE /v1/api-keys/:id`

并自动注册 `api_key` 中间件组与 `key` alias，业务模块可以直接使用：

```go
r.Group("/v1", func(api *router.Router) {
    api.WithMiddleware("api_key")
    api.GET("/inference", handler.Run)
})
```

## 可选集成

这些能力保留在仓库中，但都应该被视为可选基础设施，而不是脚手架默认业务身份：

- `Redis`
- `Sentry`
- `OpenTelemetry`
- `Resend` 邮件 capability：10 秒默认 provider timeout、64 KiB 响应上限和 context 取消；边界见 [docs/EMAIL.md](docs/EMAIL.md)
- `R2` 对象存储

如果你的项目不需要这些能力，可以只保留核心 HTTP、配置、数据库、路由和模块层。

## 部署

仓库提供 production-oriented API image、本地开发 Compose 和独立容器 smoke CI。完整契约见
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)。`docker-compose.yml` 只用于本地开发，不是生产部署清单：

```bash
docker compose up --build --wait
docker compose down
```

构建配置：

- `Dockerfile`：多阶段构建，runtime 使用 distroless non-root，不内嵌任何 `.env`
- `.dockerignore`：排除本地 binary、`*.test`、日志、覆盖率和开发资料
- `health:check`：镜像内置的 loopback liveness probe，不依赖 shell/curl
- `LOG_STDOUT=true` + `LOG_FILE_ENABLED=false`：容器请求日志输出 JSON 到 stdout
- `make container-check`：真实构建、启动、探针、日志、env 泄漏和 SIGTERM 验证
- `make compose-check`：真实 PostgreSQL、启动迁移、readiness 和 starter 注册验证
- `.github/workflows/container.yml`：API/container 变更时执行相同 smoke test
- 可选镜像发布工作流可以使用 buildx + `cache-from/to: type=gha` 共享 Docker 层缓存
- 可选平台部署需要配置对应平台的 API token Secret

本轮本地 Docker Desktop 基线中，修复前构建上下文为 `40.99 MB`，修复后首次完整传输为
`87.65 kB`；镜像从 `24,942,104` bytes 变为 `24,944,318` bytes（增加 2,214 bytes，约
0.009%），换取内置健康检查和安全运行契约。这是本机 build evidence，不是跨平台镜像预算。
是否公开镜像、使用哪个 registry、如何注入 secrets、是否自动部署，均由具体项目决定。

## 设计原则

- 根仓库只表达脚手架能力，不表达具体业务产品
- 模块边界清晰，优先保证可替换和可测试
- 默认配置最小化，额外能力显式开启
- 框架自身必须遵守自己定义的模块规范

## License

MIT
