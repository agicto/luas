# Workflow Queue Capability

The workflow queue is a reusable technical capability. It remains independent of product modules
and supports both lightweight local execution and PostgreSQL-backed durable workers.

The canonical implementation lives in `internal/capabilities/workflow`. The
`internal/infra/queue` package is a compatibility wrapper only.

## Driver Matrix

| Driver | Execution | Persistence | Intended use |
|---|---|---|---|
| `sync` | Runs the job in the caller | None | Local development, tests, simple inline work |
| `memory` | Bounded FIFO consumed by workers | Process memory only | Local async development and single-process prototypes |
| `postgres` | Leased worker execution | PostgreSQL | Durable production tasks and horizontally scaled workers |

Both drivers honor cancellation while waiting for a delayed dispatch. The memory driver also honors
context cancellation while blocked on a full or empty queue.

## Memory Driver Contract

- `NewMemoryDriver(n)` creates a bounded FIFO for each queue name. Values below one are normalized
  to a capacity of one.
- Pending delayed deliveries share a driver-wide budget of `n`. When that budget is full,
  `PushDelayed` applies backpressure until a slot opens, its context is canceled, or the driver
  closes. This prevents unbounded timer goroutine growth.
- A successful `Push` or `PushDelayed` transfers ownership of the payload slice to the driver. The
  caller must not modify or reuse that slice afterward. `QueueManager` dispatches freshly serialized
  payloads and already satisfies this rule.
- `PushDelayed` is process-local scheduling. A request-scoped context cancels the pending delivery
  when that request ends; use an application-lifecycle context when work should outlive a request.
- `Clear` removes jobs present at its synchronization point. Concurrent pushes that complete later
  may add new jobs after `Clear` returns.
- `Close` is concurrent-safe and idempotent. It rejects new operations, cancels pending delayed
  deliveries, wakes blocked producers and consumers, waits for in-flight operations to exit, then
  releases queued payloads.
- Every operation started after closure returns `ErrDriverClosed`. A worker exits when its driver is
  closed.

The memory driver does not provide durability, cross-process coordination, acknowledgements,
visibility timeouts, a dead-letter queue, or recovery after process termination. Delivery is
at-most-once after `Pop`: a process crash can lose queued or running work.

## PostgreSQL Driver Contract

Set `QUEUE_DRIVER=postgres`, run migrations, and start one or more workers with
`luas workflow:work`. Producers and workers may run in different replicas.

- Dispatch writes a `workflow_tasks` ledger row before returning.
- Delayed tasks become claimable at `available_at`.
- `FOR UPDATE SKIP LOCKED` allows replicas to claim distinct tasks without a coordinator.
- Every claim receives a random lease token and an increasing fencing token. Completion, retry,
  failure, and cancellation reject stale ownership.
- Heartbeats renew active leases and deliver cooperative cancellation through the job context.
- Expired leases are reclaimed. A final expired attempt moves to `failed` for dead-letter
  inspection instead of remaining stuck in `processing`.
- Jobs implementing `JobWithIdempotencyKey` are unique within a queue. Reusing a key with the same
  semantic payload succeeds; reuse with different content returns `ErrIdempotencyConflict`.
- OpenTelemetry `traceparent` and `tracestate` values are stored with the serialized job and restored
  before execution.
- `workflow_queue_tasks{queue,state}` and `workflow_queue_lag_seconds{queue}` expose bounded queue
  depth, failure, active-work, and lag signals.

PostgreSQL remains the only relational database target. The durable driver requires neither Redis
nor a separate broker.

```bash
QUEUE_DRIVER=postgres ./bin/luas migrate
QUEUE_DRIVER=postgres ./bin/luas workflow:work --queue=default --concurrency=4
```

Keep `QUEUE_LEASE_DURATION` greater than `QUEUE_WORKER_TIMEOUT`. The heartbeat interval must be
shorter than the lease duration. Handler side effects still need their own idempotency key because
reliable queues provide at-least-once execution around process failure.

## Worker Lifecycle

`Worker.Stop` interrupts an empty-queue wait and retry sleep, cancels the worker run context, and
waits for worker goroutines to exit. `Stop` can be called more than once. A typical shutdown order is:

1. Stop accepting new dispatch requests.
2. Cancel the application or worker context.
3. Call `Worker.Stop` and wait for active workers.
4. Call `Driver.Close` to cancel delayed deliveries and release memory.

The CLI worker already uses a signal-aware parent context for `SIGINT` and `SIGTERM`.

## Custom Drivers

Downstream applications may implement `workflow.Driver` or `workflow.DurableDriver` for a
specialized broker. Preserve these observable behaviors:

- context-aware push, delayed push, and pop operations;
- stable closed-driver errors;
- idempotent resource closure;
- explicit retry, acknowledgement, and dead-letter semantics;
- queue depth and failure observability without high-cardinality labels.

Do not present `QUEUE_DRIVER=memory` as horizontally scalable. Every API or worker process owns an
independent queue, so producers and consumers in different replicas cannot see one another's jobs.

## Verification

```bash
make test-race-critical
make benchmark-workflow
LUAS_TEST_POSTGRES_DSN='postgres://...' go test ./internal/capabilities/workflow -run Postgres
go test ./...
```
