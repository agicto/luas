# API Key Browser Feature

The default API key feature provides a real management surface at `/console/settings` and browser
routes under `/api/api-keys`. Its shared HTTP and security behavior is defined in
[`../../contracts/API_KEYS.md`](../../contracts/API_KEYS.md).

## Runtime Resolution

| Runtime | Backend |
|---|---|
| `API_ADAPTER_ENABLED=true` | Fixed server-only adapter to Go `api-keys` paths |
| Development, adapter off | In-memory mock store after mock BFF availability checks |
| Production, adapter and mock off | `503 COMMON.SERVICE_UNAVAILABLE` |

The adapter owns the HttpOnly API session token and does not accept a browser-selected upstream
path. Create and revoke apply the same-origin mutation guard before authentication and body parsing.
Every route is finalized as `private, no-store` and varies on `Cookie`.

## Secret Lifecycle

The create mutation validates the successful response, copies metadata into the normal list only by
refetching, and keeps `plaintext_key` in dialog-local state. It configures mutation garbage
collection with `gcTime: 0`, calls `reset()` immediately after extracting the result, and clears the
dialog state on close. List schemas are strict and reject either `plaintext_key` or `key_hash`.

The mock store follows the same rule: it returns plaintext once, stores only a SHA-256 hash plus
metadata, and never includes the hash in public list data.

## Downstream Changes

- Keep the feature when the product uses user-owned API keys and its scope grammar is suitable.
- Replace the fixed routes and update the shared contract when another backend has different paths
  or DTOs; do not turn the adapter into a catch-all proxy.
- Remove `src/features/api-key`, `src/app/api/api-keys`, the API tab, its route-owned translations,
  and API key tests together when the downstream product does not expose API keys.
- Define product scope names in the downstream app. Do not add a fake global scope catalog to Luas.

## Verification

```bash
pnpm vitest run src/test/api-key-contract.test.ts src/test/api-key-route.test.ts src/test/api-key-ui.test.tsx
pnpm type-check
pnpm lint
pnpm build
```
