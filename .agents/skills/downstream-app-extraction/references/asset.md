# Asset Retention

Read this reference only when the downstream app retains the optional `asset`
starter. Also read `contracts/ASSETS.md`, `api/docs/ASSETS.md`, and
`web/docs/ASSETS.md`.

- Keep user ownership and lifecycle in the starter while provider byte
  operations remain behind the storage capability.
- Run `asset:prune` with the same database, provider secrets, and
  `OPTIONAL_STARTERS` selection as API replicas.
- Preserve staging-to-final promotion, bounded inspection, the account-deletion
  guard, signed-grant privacy, exact R2 CORS, and provider lifecycle cleanup.
- Keep grants out of persistent browser state.
- Never expose object keys, local paths, checksums, or a generic bucket API.
