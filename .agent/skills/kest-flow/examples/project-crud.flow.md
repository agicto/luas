```flow
@flow id=project-crud
@name Project CRUD Flow
@version 1.0
@tags project, crud
@env dev
```

```step
@id create-project
@name Create Project
@retry 2

POST /api/v1/projects
Content-Type: application/json
Authorization: Bearer {{token}}

{
  "name": "Test Project {{$randomInt}}",
  "description": "A test project",
  "visibility": "private"
}

[Captures]
project_id = data.id

[Asserts]
status == 201
body.data.id exists
```

```step
@id get-project
@name Get Project

GET /api/v1/projects/{{project_id}}
Authorization: Bearer {{token}}

[Asserts]
status == 200
body.data.id == {{project_id}}
```

```step
@id update-project
@name Update Project

PUT /api/v1/projects/{{project_id}}
Content-Type: application/json
Authorization: Bearer {{token}}

{
  "name": "Updated Project",
  "description": "Updated description"
}

[Asserts]
status == 200
```

```step
@id list-projects
@name List Projects

GET /api/v1/projects
Authorization: Bearer {{token}}

[Asserts]
status == 200
body.data array_not_empty
```

```step
@id delete-project
@name Delete Project

DELETE /api/v1/projects/{{project_id}}
Authorization: Bearer {{token}}

[Asserts]
status == 204
```

```edge
@from create-project
@to get-project
@on success
```

```edge
@from get-project
@to update-project
@on success
```

```edge
@from update-project
@to list-projects
@on success
```

```edge
@from list-projects
@to delete-project
@on success
```
