```flow
@flow id=zgo-api-keys-starter
@name ZGO API Key Starter Flow
@version 1.0
@tags apikey, starter
@env local
```

```step
@id register
@name Register User
@retry 2

POST /v1/register
Content-Type: application/json

{
  "username": "apikey_user_{{run_id}}",
  "email": "apikey_{{run_id}}@example.com",
  "password": "password123",
  "nickname": "API Key User"
}

[Captures]
email = data.email

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
  "username": "{{email}}",
  "password": "password123"
}

[Captures]
token = data.access_token

[Asserts]
status == 200
body.data.access_token exists
```

```step
@id create-key
@name Create API Key

POST /v1/api-keys
Content-Type: application/json
Authorization: Bearer {{token}}

{
  "name": "Automation Key",
  "scopes": ["models:invoke", "models:read"]
}

[Captures]
api_key_id = data.api_key.id
plaintext_key = data.plaintext_key

[Asserts]
status == 201
body.data.api_key.id exists
body.data.plaintext_key exists
```

```step
@id list-keys
@name List API Keys

GET /v1/api-keys
Authorization: Bearer {{token}}

[Asserts]
status == 200
body.data.0.id exists
body.data.0.name exists
```

```step
@id revoke-key
@name Revoke API Key

DELETE /v1/api-keys/{{api_key_id}}
Authorization: Bearer {{token}}

[Asserts]
status == 204
```

```edge
@from register
@to login
@on success
```

```edge
@from login
@to create-key
@on success
```

```edge
@from create-key
@to list-keys
@on success
```

```edge
@from list-keys
@to revoke-key
@on success
```
