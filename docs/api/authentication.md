# Authentication API

Trac API 支持两种认证方式：
1. **JWT Token 认证** - 用于管理 API（项目、用户、Issue 管理）
2. **Public Key 认证** - 用于 SDK 事件上报

---

## 1. JWT Token 认证

### 用户注册

#### POST `/v1/register`

创建新用户账号。

**请求体:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "secure_password_123",
  "nickname": "John Doe"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": 3,
    "username": "john_doe",
    "email": "john@example.com",
    "nickname": "John Doe",
    "status": 1,
    "created_at": "2026-01-13T02:47:14Z",
    "updated_at": "2026-01-13T02:47:14Z"
  }
}
```

---

### 用户登录

#### POST `/v1/login`

使用用户名和密码登录，获取 JWT Token。

**请求体:**
```json
{
  "username": "john_doe",
  "password": "secure_password_123"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 3,
      "username": "john_doe",
      "email": "john@example.com",
      "nickname": "John Doe",
      "status": 1,
      "last_login": "2026-01-13T02:47:14Z",
      "created_at": "2026-01-13T02:47:14Z",
      "updated_at": "2026-01-13T02:47:14Z"
    }
  }
}
```

**Token 有效期:** 7 天（可通过环境变量 `JWT_EXPIRE_DAYS` 配置）

---

### 使用 JWT Token

所有需要认证的 API 都需要在请求头中携带 JWT Token：

```bash
curl -X GET http://your-server:8025/v1/projects \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**需要认证的 API:**
- 所有 `/v1/projects/*` 接口
- 所有 `/v1/users/*` 接口（除了 `/register` 和 `/login`）
- 所有 `/v1/permissions/*` 接口

---

## 2. Public Key 认证（SDK 事件上报）

SDK 使用项目的 `public_key` 进行认证，无需 JWT Token。

### 获取 Public Key

通过项目管理 API 获取：

```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST http://your-server:8025/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"pass"}' | jq -r '.data.access_token')

# 2. 获取项目 DSN（包含 public_key）
curl -s http://your-server:8025/v1/projects/1/dsn \
  -H "Authorization: Bearer $TOKEN" | jq
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "dsn": "http://4fbf7efef9b5b1d067dfac36d1cd891f@your-server:8025/1",
    "project_id": 1,
    "public_key": "4fbf7efef9b5b1d067dfac36d1cd891f"
  }
}
```

---

### SDK 认证方式

#### 方式 1: 使用 DSN（推荐）

```go
// Go SDK
sentry.Init(sentry.ClientOptions{
    Dsn: "http://4fbf7efef9b5b1d067dfac36d1cd891f@your-server:8025/1",
})
```

```javascript
// JavaScript SDK
Sentry.init({
  dsn: "http://4fbf7efef9b5b1d067dfac36d1cd891f@your-server:8025/1",
});
```

#### 方式 2: 使用 X-Sentry-Auth Header

```bash
curl -X POST http://your-server:8025/api/1/envelope/ \
  -H "X-Sentry-Auth: Sentry sentry_version=7, sentry_key=4fbf7efef9b5b1d067dfac36d1cd891f" \
  --data-binary @envelope.txt
```

---

## 错误处理

### 401 Unauthorized

**原因:**
- JWT Token 缺失或无效
- Token 已过期
- Public Key 不正确

**响应示例:**
```json
{
  "code": 401,
  "message": "unauthorized",
  "error": "invalid or expired token"
}
```

**解决方案:**
1. 检查 `Authorization` header 格式是否正确
2. 重新登录获取新的 Token
3. 验证 Public Key 是否匹配项目

---

## 安全建议

1. **HTTPS**: 生产环境务必使用 HTTPS 传输 Token
2. **Token 存储**: 不要在客户端明文存储 Token
3. **Public Key 保护**: 不要在公开代码中硬编码 Public Key
4. **定期轮换**: 定期更新项目的 Public Key（未来功能）

---

## 示例：完整认证流程

```bash
#!/bin/bash

# 1. 注册用户
curl -X POST http://localhost:8025/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "demo_user",
    "email": "demo@example.com",
    "password": "demo123456",
    "nickname": "Demo User"
  }'

# 2. 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8025/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo_user","password":"demo123456"}' \
  | jq -r '.data.access_token')

echo "Token: $TOKEN"

# 3. 使用 Token 访问受保护的 API
curl -X GET http://localhost:8025/v1/projects \
  -H "Authorization: Bearer $TOKEN"

# 4. 创建项目并获取 DSN
PROJECT=$(curl -s -X POST http://localhost:8025/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My App","platform":"go"}')

DSN=$(echo "$PROJECT" | jq -r '.data.dsn')
echo "DSN: $DSN"

# 5. 使用 DSN 配置 SDK
# 在应用代码中使用此 DSN
```
