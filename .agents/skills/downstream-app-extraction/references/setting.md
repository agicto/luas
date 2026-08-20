# Setting Retention

Read this reference only when the downstream app retains the optional `setting`
starter.

- Retain `organization` in both API and Web optional selections.
- Keep the catalog finite, code-owned, scalar, and typed. Extend definitions at
  assembly time; do not add HTTP definition creation or a generic JSON editor.
- Preserve strong `If-Match` writes, monotonic reset tombstones, public app
  ETags, private no-store responses, value-free audit metadata, user cleanup,
  and strict Web definition validation.
- Keep secrets, process configuration, permissions, entitlements, usage limits,
  and notification preferences with their owning seams.
