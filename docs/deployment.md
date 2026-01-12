# 项目配置指南

## 环境变量配置

### 服务器配置

```bash
# 服务端口
SERVER_PORT=8025

# 运行模式
GIN_MODE=debug  # debug | release | test

# CORS 配置
CORS_ALLOW_ORIGINS=*
CORS_ALLOW_METHODS=GET,POST,PUT,DELETE,OPTIONS
```

### PostgreSQL 配置

```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=trac
DB_USERNAME=postgres
DB_PASSWORD=your_password

# 连接池配置
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=3600
```

### ClickHouse 配置

```bash
# 启用 ClickHouse
LOG_CH_ENABLED=true

# 连接配置
LOG_CH_ENDPOINT=localhost:9000
LOG_CH_DATABASE=trac
LOG_CH_USERNAME=trac_user
LOG_CH_PASSWORD=trac_pass

# 批量写入配置
LOG_CH_BATCH_SIZE=100
LOG_CH_INTERVAL=5s
```

### JWT 配置

```bash
JWT_SECRET=your-super-secret-key-at-least-32-chars
JWT_EXPIRE_HOURS=24
```

---

## Docker 部署

### docker-compose.yml

```yaml
version: '3.8'

services:
  trac-api:
    build: .
    ports:
      - "8025:8025"
    environment:
      - SERVER_PORT=8025
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=trac
      - DB_USERNAME=postgres
      - DB_PASSWORD=postgres
      - LOG_CH_ENABLED=true
      - LOG_CH_ENDPOINT=clickhouse:9000
      - LOG_CH_DATABASE=trac
      - LOG_CH_USERNAME=trac_user
      - LOG_CH_PASSWORD=trac_pass
    depends_on:
      - postgres
      - clickhouse

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: trac
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    environment:
      CLICKHOUSE_DB: trac
      CLICKHOUSE_USER: trac_user
      CLICKHOUSE_PASSWORD: trac_pass
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    ports:
      - "9000:9000"
      - "8123:8123"

volumes:
  postgres_data:
  clickhouse_data:
```

### 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f trac-api

# 停止服务
docker-compose down
```

---

## 数据库初始化

### PostgreSQL

服务启动时会自动运行 GORM 迁移，创建以下表：

- `users` - 用户表
- `projects` - 项目表

### ClickHouse

服务启动时会自动创建：

- `events` - 事件表（MergeTree）
- `issues` - Issue 聚合表（AggregatingMergeTree）
- `issues_mv` - 物化视图（自动聚合）

---

## 健康检查

```bash
# 检查服务状态
curl http://localhost:8025/health

# 响应
{
  "status": "up",
  "checks": {
    "database": {"status": "up"}
  }
}
```

---

## 生产环境建议

### 安全配置

```bash
# 使用 HTTPS
# 在反向代理（Nginx/Caddy）配置 TLS

# 设置强密码
JWT_SECRET=<random-64-char-string>
DB_PASSWORD=<strong-password>
LOG_CH_PASSWORD=<strong-password>

# 限制 CORS
CORS_ALLOW_ORIGINS=https://your-domain.com
```

### 性能优化

```bash
# 增加连接池
DB_MAX_OPEN_CONNS=200
DB_MAX_IDLE_CONNS=50

# 调整批量写入
LOG_CH_BATCH_SIZE=500
LOG_CH_INTERVAL=10s
```

### 日志配置

```bash
# 结构化日志
LOG_FORMAT=json
LOG_LEVEL=info

# 日志输出
LOG_OUTPUT=stdout
```
