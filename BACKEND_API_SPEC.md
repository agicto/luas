# Backend API Interaction Specification (Frontend Perspective)

This document outlines the protocol for backend integration to ensure full end-to-end type safety and optimal developer experience.

## 1. Schema-First Development

The frontend uses **Type-only Synchronization**. We do not manually define API data structures.

- **Requirement**: Backend must provide a valid **OpenAPI v3 (Swagger)** JSON/YAML specification.
- **Auto-Sync**: The frontend runs `pnpm gen:api` to synchronize TypeScript interfaces with the latest backend schema.
- **Breaking Changes**: Any change to the backend schema will be immediately visible to the frontend compiler.

## 2. Standard Response Envelope

We prefer a consistent response structure across all endpoints.

### Success Response
```json
{
  "data": { ... },
  "message": "Operation successful",
  "total": 100 // For list endpoints
}
```

### Error Response
We use the global error handler to map these to UI notifications.
```json
{
  "error": "Short error code",
  "message": "User-facing descriptive message",
  "details": { ... }
}
```

## 3. Authentication (Bearer Token)

- **Mechanism**: The frontend sends the token in the `Authorization` header.
- **Format**: `Bearer <token>`
- **Refresh Flow**: If the backend supports `RefreshToken`, the frontend interceptor will handle the silent retry logic before the user perceives a failure.

## 4. Status Codes

- `200/201`: Success
- `401`: Unauthorized (Trigger global logout/redirect)
- `403`: Forbidden (Show permission toast)
- `422`: Validation Error (Map to form field errors)
- `500`: System Error (Show generic retry toast)

## 5. Metadata & Enums

- All enums should be defined in the OpenAPI spec to ensure TypeScript autocomplete.
- Date strings should follow ISO 8601 format.
