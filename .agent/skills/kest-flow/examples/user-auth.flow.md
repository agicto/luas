```flow
@flow id=user-auth
@name User Authentication Flow
@version 1.0
@tags auth, user
@env local
```

```step
@id register
@name Register User
@retry 2

POST /v1/register
Content-Type: application/json

{
  "username": "testuser{{$randomInt}}",
  "email": "test{{$randomInt}}@example.com",
  "password": "password123",
  "nickname": "Test User"
}

[Captures]
user_id = data.id
username = data.username

[Asserts]
status == 201
body.data.id exists
```

```step
@id login
@name Login
@retry 2

POST /v1/login
Content-Type: application/json

{
  "username": "{{username}}",
  "password": "password123"
}

[Captures]
token = data.access_token

[Asserts]
status == 200
body.data.access_token exists
```

```step
@id profile
@name Get Profile

GET /v1/users/profile
Authorization: Bearer {{token}}

[Asserts]
status == 200
body.data.username exists
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
