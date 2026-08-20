# Notification Retention

Read this reference only when the downstream app retains the optional
`notification` starter.

- Keep publication inside trusted API services and derive stable idempotency
  keys from business operations.
- Run `notification:work` with the same database, email secrets, image, and
  `OPTIONAL_STARTERS` selection as API replicas.
- Keep notification content plain text, action URLs local, and browser state
  user-scoped.
- Keep provider and recipient details outside contracts, audit records, and
  logs.
- Define retention before enabling high-volume use.
