```flow
@flow id=zgo-auth-starter
@name ZGO Auth Starter Flow
@version 1.0
@tags auth, starter
@env local
```

```step
@id register
@name Register User
@retry 2

POST /v1/register
Content-Type: application/json

{
  "username": "kest_user_{{run_id}}",
  "email": "kest_{{run_id}}@example.com",
  "password": "password123",
  "nickname": "Kest User"
}

[Captures]
user_id = data.id
email = data.email

[Asserts]
status == 201
body.data.id exists
body.data.email exists
```

```step
@id login
@name Login With Email
@retry 2

POST /v1/login
Content-Type: application/json

{
  "username": "{{email}}",
  "password": "password123"
}

[Captures]
token = data.access_token

[Asserts]
status == 200
body.data.access_token exists
body.data.user.id exists
```

```step
@id profile
@name Fetch Profile

GET /v1/users/profile
Authorization: Bearer {{token}}

[Asserts]
status == 200
body.data.id exists
body.data.email exists
```

```edge
@from register
@to login
@on success
```

```edge
@from login
@to profile
@on success
```
