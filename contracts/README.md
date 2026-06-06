# Luas Contracts

This directory is the explicit interface between `api/` and `web/`.

The two halves of Luas do not share runtime code. API behavior is exposed over HTTP, and the web app depends on documented request, response, and error shapes. Keep contracts small, stable, and easy to review.

## Contract Rules

- Write contracts before wiring a new cross-service feature.
- Use `luas make:contract <name>` to create the starter contract shape.
- Keep examples in JSON, not framework-specific code.
- Name fields in `snake_case` on the wire.
- Document the success shape, validation errors, auth requirements, and pagination rules together.
- Update the matching web service and API handler in the same change when a contract changes.
- Run `luas map --json` after contract changes so AI agents can see the updated interface index.

## Standard Shapes

Use these shapes unless a feature has a documented reason to differ.

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

```json
{
  "code": 422,
  "error_code": "COMMON.VALIDATION_FAILED",
  "message": "Validation failed",
  "errors": {},
  "request_id": "req_..."
}
```

```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "meta": {},
  "links": {}
}
```

## Recommended File Layout

```text
contracts/
├── README.md
├── auth.md
├── users.md
├── apikey.md
├── audit.md
├── team.md
└── access.md
```

Start with a Markdown contract. Add OpenAPI later only when the contract volume justifies generated tooling.
