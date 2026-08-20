# Mock BFF And Authentication Replacement

Read this reference only when a downstream extraction deletes, replaces, or
retains mock BFF or authentication adapter behavior.

- Replace mock BFF routes with production endpoints or a documented
  same-origin adapter. If retained for development, preserve production and
  same-origin mutation guards.
- Read `contracts/AUTHENTICATION.md`. The Web cookie contract and Go
  authentication-session contract are not interchangeable through a base-URL
  change.
- Preserve the provider-owned authentication store.
- Keep `client-session` unless the downstream Next.js server can
  authoritatively resolve the real session; then replace only the bootstrap
  adapter.
- Keep fixed upstream paths, DTO mappings, credential forwarding, timeout and
  error translation explicit. Never introduce a browser-controlled generic
  proxy.
- Verify auth, CORS, credentials, and error envelopes through focused Web tests
  and a browser or curl check appropriate to the changed adapter.
