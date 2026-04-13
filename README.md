# ZGO

> 面向模块化 Go API 的纯脚手架

ZGO 是一个用于搭建 Go 后端项目的脚手架，目标是提供稳定的项目结构、依赖注入、模块边界、统一响应、分页、迁移、测试工具和常用基础设施集成。

这个仓库的定位是“框架与模板”，不是某个具体业务系统。默认只保留最小可用的认证与 API key starter，增强型业务模块和示例能力不再自动挂载到主应用。

## 核心能力

- 模块化目录结构，适合 DDD + 分层架构
- Gin HTTP 服务入口与统一路由注册
- Wire 依赖注入
- GORM 数据访问与迁移体系
- 内置 starter：`user`, `apikey`
- Provider-neutral AI capability 与内置 CLI `ai:chat`
- 统一 API 响应与错误处理
- 分页、验证、日志、JWT、中间件
- 测试辅助工具与集成测试基线
- 可选集成：Redis、邮件、OpenTelemetry、ClickHouse 日志通道、Sentry

## 快速开始

### 1. 环境准备

- Go 1.24+
- PostgreSQL 12+ 或 SQLite
- Redis 6+（可选）

### 2. 初始化配置

```bash
cp .env.example .env
```

最少需要确认以下配置：

```bash
APP_NAME=ZGO
APP_ENV=development
SERVER_PORT=8025

DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_NAME=zgo

JWT_SECRET=replace-me
```

### 3. 生成依赖注入代码

```bash
make wire
```

### 4. 启动 HTTP 服务

```bash
go run ./cmd/server
```

默认地址：

- 应用首页：`http://localhost:8025/`
- 健康检查：`http://localhost:8025/v1/health`
- Swagger：`http://localhost:8025/swagger/index.html`

### 5. 使用 CLI

```bash
go run ./cmd/zgo version
go run ./cmd/zgo route:list
go run ./cmd/zgo migrate
go run ./cmd/zgo seed
go run ./cmd/zgo ai:chat "Summarize this scaffold in one sentence"
go run ./cmd/zgo deploy:targets
go run ./cmd/zgo deploy:run --target=api-local --branch=main
go run ./cmd/zgo deploy:list
go run ./cmd/zgo deploy:logs <deployment-id>
go run ./cmd/zgo deploy:cert --domain=api.example.com
```

## 自动化发布系统

仓库现在内置了一套轻量发布编排能力，覆盖：

- `CLI` 触发发布
- `HTTP API` 触发发布、查历史、看日志、SSE 实时日志流
- 文件落盘的发布记录与日志归档
- 自签名证书生成
- Render 兼容目标配置示例
- 平台控制面：GitHub 连接、项目/服务、环境变量、仓库驱动部署、GitHub push 自动部署

默认配置文件：

```text
deploy.targets.json
deploy.targets.render.example.json
storage/deployments/
```

CLI 示例：

```bash
go run ./cmd/zgo deploy:targets
go run ./cmd/zgo deploy:run --target=fullstack-local --branch=main
go run ./cmd/zgo deploy:list --limit=10
go run ./cmd/zgo deploy:logs <deployment-id> --tail=200
go run ./cmd/zgo deploy:cert --domain=app.example.com --days=90
```

HTTP 端点：

- `GET /v1/deploy/targets`
- `GET /v1/deployments`
- `POST /v1/deployments`
- `GET /v1/deployments/:id`
- `GET /v1/deployments/:id/logs`
- `GET /v1/deployments/:id/stream`
- `POST /v1/deployments/certificates`
- `POST /v1/deployments/webhooks/:target`

平台端点：

- `GET /v1/platform/overview`
- `GET /v1/platform/deploy-targets`
- `GET /v1/platform/github/connections`
- `POST /v1/platform/github/connections`
- `GET /v1/platform/github/connections/:id/repositories`
- `GET /v1/platform/projects`
- `POST /v1/platform/projects`
- `GET /v1/platform/services`
- `POST /v1/platform/services/import`
- `GET /v1/platform/services/:id`
- `PUT /v1/platform/services/:id`
- `PUT /v1/platform/services/:id/environment`
- `GET /v1/platform/services/:id/deployments`
- `POST /v1/platform/services/:id/deploy`
- `POST /v1/platform/services/:id/webhooks/github`
- `GET /v1/platform/deployments/:deploymentId/logs`
- `GET /v1/platform/deployments/:deploymentId/stream`

平台环境变量：

```bash
DEPLOY_TARGETS_PATH=deploy.targets.json
DEPLOY_STORAGE_ROOT=storage/deployments
PLATFORM_STORAGE_ROOT=storage/platform
GITHUB_API_BASE_URL=https://api.github.com
```

平台导入流程：

1. `POST /v1/platform/github/connections` 绑定 GitHub Token
2. `GET /v1/platform/github/connections/:id/repositories` 拉取仓库
3. `POST /v1/platform/services/import` 选择仓库并创建服务
4. `POST /v1/platform/services/:id/deploy` 触发部署
5. `GET /v1/platform/deployments/:deploymentId/stream` 看实时日志
6. 可选：把 GitHub push webhook 指向 `POST /v1/platform/services/:id/webhooks/github`

Webhook 自动发布约定：

- 在目标配置里声明 `autoDeployBranches`
- 如果配置了 `webhookSecret`，请求头带 `X-Deploy-Secret`
- 请求体最少包含 `branch`

```json
{
  "branch": "main",
  "commit": "abc123",
  "triggeredBy": "github-actions"
}
```

## 常用命令

```bash
make build
make test
make lint
make wire
make air
```

## 项目结构

```text
zgo/
├── cmd/
│   ├── server/               # HTTP 服务入口
│   └── zgo/                  # CLI 入口
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
- `internal/modules/permission` 保留为可选 RBAC 示例模块，不再默认装配到主应用

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
AI_DEFAULT_MODEL=gpt-5.4
OPENAI_API_KEY=replace-me
```

命令示例：

```bash
go run ./cmd/zgo ai:chat "Write a short project summary"
go run ./cmd/zgo ai:chat --system="Answer in JSON" --model=gpt-5.4 "List 3 scaffold priorities"
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
- `ClickHouse` 日志输出
- `OpenTelemetry`
- `Resend` 邮件服务
- `R2` 对象存储

如果你的项目不需要这些能力，可以只保留核心 HTTP、配置、数据库、路由和模块层。

## 设计原则

- 根仓库只表达脚手架能力，不表达具体业务产品
- 模块边界清晰，优先保证可替换和可测试
- 默认配置最小化，额外能力显式开启
- 框架自身必须遵守自己定义的模块规范

## License

MIT
