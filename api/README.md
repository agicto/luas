# ZGO

> 面向模块化 Go API 的纯脚手架

ZGO 是一个用于搭建 Go 后端项目的脚手架，目标是提供稳定的项目结构、依赖注入、模块边界、统一响应、分页、迁移、测试工具和常用基础设施集成。

这个仓库的定位是“框架与模板”，不是某个具体业务系统。默认只保留最小可用的认证、API key 和审计 starter，增强型业务模块和示例能力不再自动挂载到主应用。

## 核心能力

- 模块化目录结构，适合 DDD + 分层架构
- Gin HTTP 服务入口与统一路由注册
- Wire 依赖注入
- GORM 数据访问与迁移体系
- 内置 starter：`user`, `apikey`, `audit`
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
- `internal/modules/audit` 是默认审计 starter，会记录全局写请求，并提供当前用户的审计历史查询
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

## 部署

仓库使用 GitHub Actions + GHCR 出镜像，Zeabur 拉镜像跑容器。`git push` 到 `main` 后整套链路全自动，端到端约 20 秒：

```
git push (main)
  → GH Actions: docker build + push ghcr.io/zgiai/zgo:sha-<short>  (≈20s, 热缓存)
  → GraphQL: updateServiceImage(serviceID, environmentID, tag)
  → Zeabur 节点: docker pull (≈2s) + 滚动重启 (≈2s)
  → 新容器对外提供服务
```

构建配置：

- `Dockerfile`：多阶段构建，runtime 用 `gcr.io/distroless/static-debian12:nonroot`，最终镜像约 92MB
- `.dockerignore`：排除 `.git`、`tmp`、`docs`、`tests` 等无关目录
- `.github/workflows/build-image.yml`：buildx + `cache-from/to: type=gha` 共享 Docker 层缓存
- `ZEABUR_TOKEN` Secret：Zeabur GraphQL API 调用凭证

镜像在 GHCR 是 public 的，编译产物可公开；源码仓库仍为 private。

## 设计原则

- 根仓库只表达脚手架能力，不表达具体业务产品
- 模块边界清晰，优先保证可替换和可测试
- 默认配置最小化，额外能力显式开启
- 框架自身必须遵守自己定义的模块规范

## License

MIT
