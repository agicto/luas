# Error Codes Reference

Trac API 使用标准 HTTP 状态码和自定义错误码来表示请求结果。

---

## HTTP 状态码

| 状态码 | 说明 | 常见场景 |
|--------|------|----------|
| 200 | OK | 请求成功 |
| 201 | Created | 资源创建成功 |
| 400 | Bad Request | 请求参数错误 |
| 401 | Unauthorized | 未授权或认证失败 |
| 403 | Forbidden | 权限不足 |
| 404 | Not Found | 资源不存在 |
| 409 | Conflict | 资源冲突（如重复创建） |
| 422 | Unprocessable Entity | 请求格式正确但语义错误 |
| 429 | Too Many Requests | 超过速率限制 |
| 500 | Internal Server Error | 服务器内部错误 |
| 503 | Service Unavailable | 服务暂时不可用 |

---

## 响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    // 响应数据
  }
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "Invalid request parameters",
  "error": "Key: 'UserRegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
}
```

---

## 自定义错误码

### 认证相关 (401)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `missing authentication` | 缺少认证信息 | 添加 Authorization header 或 X-Sentry-Auth |
| `invalid or expired token` | Token 无效或已过期 | 重新登录获取新 Token |
| `invalid username or password` | 用户名或密码错误 | 检查登录凭据 |
| `invalid key` | Public Key 无效 | 检查项目 DSN 中的 public_key |

### 资源相关 (404)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `project not found` | 项目不存在 | 检查项目 ID 是否正确 |
| `user not found` | 用户不存在 | 检查用户 ID 是否正确 |
| `issue not found` | Issue 不存在 | 检查 fingerprint 是否正确 |

### 权限相关 (403)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `project disabled` | 项目已禁用 | 联系管理员启用项目 |
| `permission denied` | 权限不足 | 检查用户权限 |

### 参数验证 (400)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `Invalid request parameters` | 请求参数错误 | 检查请求体格式和必填字段 |
| `invalid envelope` | Envelope 格式错误 | 检查事件上报格式 |
| `payload too large` | 请求体过大 | 减小请求体大小（最大 40MB） |

### 业务逻辑 (409, 422)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `username already exists` | 用户名已存在 | 使用不同的用户名 |
| `email already exists` | 邮箱已被注册 | 使用不同的邮箱或找回密码 |
| `project name already exists` | 项目名已存在 | 使用不同的项目名 |

### 速率限制 (429)

| 错误信息 | 说明 | 解决方案 |
|---------|------|----------|
| `rate limit exceeded` | 超过速率限制 | 等待后重试，或联系管理员提高限额 |

---

## 错误处理示例

### JavaScript

```javascript
async function createProject(name, platform) {
  try {
    const response = await fetch('http://your-server:8025/v1/projects', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ name, platform })
    });
    
    const data = await response.json();
    
    if (data.code !== 0) {
      // 处理业务错误
      console.error(`Error ${data.code}: ${data.message}`);
      if (data.error) {
        console.error('Details:', data.error);
      }
      return null;
    }
    
    return data.data;
  } catch (error) {
    // 处理网络错误
    console.error('Network error:', error);
    return null;
  }
}
```

### Go

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Error   string `json:"error,omitempty"`
}

func createProject(name, platform string) error {
    // ... make request ...
    
    var errResp ErrorResponse
    if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }
    
    if errResp.Code != 0 {
        return fmt.Errorf("API error %d: %s - %s", 
            errResp.Code, errResp.Message, errResp.Error)
    }
    
    return nil
}
```

### Python

```python
import requests

def create_project(name, platform, token):
    try:
        response = requests.post(
            'http://your-server:8025/v1/projects',
            headers={'Authorization': f'Bearer {token}'},
            json={'name': name, 'platform': platform}
        )
        
        data = response.json()
        
        if data['code'] != 0:
            print(f"Error {data['code']}: {data['message']}")
            if 'error' in data:
                print(f"Details: {data['error']}")
            return None
            
        return data['data']
        
    except requests.exceptions.RequestException as e:
        print(f"Network error: {e}")
        return None
```

---

## 调试技巧

### 1. 启用详细日志

在开发环境中，设置 `APP_DEBUG=true` 可以获取更详细的错误信息。

### 2. 检查响应头

```bash
curl -v http://your-server:8025/v1/projects \
  -H "Authorization: Bearer $TOKEN"
```

### 3. 查看服务器日志

```bash
# Docker 环境
docker logs zgo-api --tail 100

# 查找错误
docker logs zgo-api 2>&1 | grep ERROR
```

### 4. 验证请求格式

使用 `jq` 验证 JSON 格式：

```bash
echo '{"name":"test","platform":"go"}' | jq .
```

---

## 常见问题排查

### Q: 为什么一直返回 401？

**检查清单:**
1. Token 是否正确复制（注意空格和换行）
2. Authorization header 格式：`Bearer <token>`
3. Token 是否已过期（默认 7 天）
4. 是否使用了正确的 API 端点

### Q: 为什么返回 404？

**检查清单:**
1. URL 路径是否正确（注意 `/v1` 前缀）
2. 资源 ID 是否存在
3. 是否有权限访问该资源

### Q: 为什么事件上报失败？

**检查清单:**
1. Public Key 是否正确
2. Project ID 是否正确
3. Envelope 格式是否符合 Sentry 规范
4. 请求体大小是否超过 40MB

---

## 获取帮助

如果遇到未在此文档中列出的错误，请：

1. 查看服务器日志获取详细信息
2. 检查 [GitHub Issues](https://github.com/zgiai/trac-api/issues)
3. 提交新的 Issue 并附上错误信息和请求详情
