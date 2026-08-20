# Usage Retention

Read this reference only when the downstream app retains the optional `usage`
starter.

- Retain `organization` in both API and Web optional selections.
- Keep metric and dimension catalogs finite and code-owned.
- Producers call domain record/consume seams with stable `source + event_id`;
  never add a public ingestion endpoint or browser quota writer.
- Preserve safe integers, UTC periods, atomic consume decisions, durable
  denials, quota CAS and tombstones, private summaries, the 90-day receipt
  horizon, pruning, and user cleanup.
- Keep telemetry, rate limits, entitlements, prices, plans, invoices, and
  provider events outside the starter.
- Run operator commands and prune jobs with the same database, image, clock
  policy, and `OPTIONAL_STARTERS` selection as API replicas.
