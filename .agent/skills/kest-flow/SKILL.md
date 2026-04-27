# Kest Flow Skill

Kest Flow is a powerful API testing framework that achieves "Documentation as Code" through Markdown files.

## When to Use

- **API Testing**: Test REST APIs with chainable flows
- **Integration Testing**: Test complete business flows (Register → Login → Create → Query)
- **Documentation**: Write executable API documentation
- **CI/CD Integration**: Integrate API tests into pipeline

## Core Concepts

### Flow File Structure

```markdown
# user.flow.md
```flow
@flow id=user-onboarding
@name User Onboarding
@version 1.0
@tags auth, user
@env dev
```

```step
@id login
@name Login
@retry 2

POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}

[Captures]
token = data.access_token

[Asserts]
status == 200
```

```edge
@from login
@to profile
@on success
```

### Variable System

**Capture from Response:**
```kest
[Captures]
token = data.access_token
user_id = data.user.id
```

**Use in Subsequent Requests:**
```kest
GET /api/v1/users/profile
Authorization: Bearer {{token}}
```

### Built-in Variables

- `{{$randomInt}}` - Random integer
- `{{$timestamp}}` - Unix timestamp

## Quick Commands

```bash
# Initialize Kest
kest init

# Run a flow
kest run user.flow.md

# Run with verbose
kest run user.flow.md -v

# Run with variable injection
kest run user.flow.md --var api_key=secret

# Parallel execution
kest run tests/ --parallel --jobs 4
```

## Examples

See `.agent/skills/kest-flow/examples/` for complete flow examples:

- `user-auth.flow.md` - User registration and login flow
- `project-crud.flow.md` - Full CRUD operations
- `hmac-signature.flow.md` - HMAC signing example

## Best Practices

1. **Use relative URLs** - Not hardcoded base URLs
2. **Chain related APIs** - Register → Login → Create → Query
3. **Add assertions** - Verify response status and data
4. **Use exec for preprocessing** - HMAC, token generation
5. **Tag flows** - `@tags auth, user` for organization
