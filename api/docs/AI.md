# AI Capability

Luas ships a provider-neutral technical capability at
[`internal/capabilities/ai`](../internal/capabilities/ai). It owns bounded provider execution. It is
not an AI product starter and does not own prompts, conversations, runs, evaluations, billing,
routes, persistence, permissions, or user workflows.

## Ownership Boundary

The capability owns:

- one-shot text generation through `Provider`;
- optional incremental text delivery through `StreamingProvider`;
- provider selection and request defaults through `Manager`;
- input, timeout, response, stream-event, redirect, and transport limits;
- stable Go error categories that do not expose provider response bodies.

A downstream AI workspace or other product module owns:

- prompt templates and version history;
- conversation and message persistence;
- durable run state, retries, cancellation, and idempotency;
- token and cost attribution, budgets, quotas, and billing;
- evaluation datasets and results;
- authorization, audit events, retention, moderation, and user-facing HTTP contracts.

Do not add those product concepts to `internal/capabilities/ai`. Build an optional starter that calls
the capability and composes the existing workflow, usage, organization, permission, and audit seams.

## Configuration

The capability is disabled by default. Enabling it requires an explicit provider, explicit provider-
owned model ID, and provider secret. Luas deliberately has no fast-moving default model.

| Key | Default | Validation when enabled |
|---|---:|---|
| `AI_ENABLED` | `false` | Must be `true` before provider calls are available |
| `AI_DEFAULT_PROVIDER` | `openai` | Must name a provider registered by this scaffold |
| `AI_DEFAULT_MODEL` | empty | Required; 1-256 bytes without whitespace or control characters |
| `AI_REQUEST_TIMEOUT` | `120s` | Greater than zero and no more than `15m` |
| `AI_MAX_INPUT_BYTES` | `1048576` | 1 KiB-16 MiB for input plus instructions |
| `AI_MAX_RESPONSE_BYTES` | `4194304` | 1 KiB-32 MiB after HTTP decompression |
| `AI_MAX_STREAM_EVENT_BYTES` | `1048576` | 1 KiB-4 MiB and no larger than the response limit |
| `OPENAI_API_KEY` | empty | Required for the built-in OpenAI provider |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | Absolute HTTP(S), no credentials/query/fragment; HTTPS in production |

`config.Load()` and the capability-scoped `config.LoadAIConfig()` enforce the same invariants. All
values are immutable for the process lifetime; changing them requires restart.

## Request Contract

`TextRequest` is intentionally small and provider-neutral:

- `Input` is required.
- `Input` plus `Instructions` must fit `AI_MAX_INPUT_BYTES` and contain valid UTF-8.
- `Provider` and `Model` are opaque identifiers, not framework enums.
- `ReasoningEffort` is an optional provider-owned identifier. Luas validates its shape but does not
  freeze a model-specific value list that would become stale.

The manager trims surrounding whitespace, applies configured defaults, validates the request, and
rejects invalid input before any provider call. Provider adapters repeat the safety validation so a
direct adapter call cannot bypass the limits.

`TextResponse.ProviderResponseID` is explicitly provider-owned. A downstream starter must generate
its own conversation, message, and run IDs rather than treating the provider identifier as a domain
identity.

## Transport And Privacy

The built-in OpenAI Responses adapter uses one reusable HTTP client with:

- caller cancellation plus a total timeout for both one-shot and complete streaming sessions;
- bounded dialing, TLS handshake, response-header wait, and 64 KiB response headers;
- TLS 1.2 minimum, HTTP/2 support, connection reuse, and standard enterprise proxy discovery;
- redirect rejection so an authenticated request cannot silently move to another endpoint;
- a 4 MiB default one-shot body cap and 1 MiB default SSE event-line cap.

Provider response bodies and provider-supplied error messages never enter returned errors. HTTP
failures return `ProviderError`, which carries only the provider name, status code, and retryable
classification. Callers can use `errors.Is(err, ErrProviderRequestFailed)` and `errors.As` instead of
parsing human text. This is a Go capability error contract, not a public HTTP `error_code` contract.

The adapter does not log prompts, instructions, generated text, API keys, or raw provider bodies.
Higher layers must preserve that privacy boundary in logs, traces, audit metadata, and exception
reporting.

## Streaming And Retry Semantics

Streaming forwards only text deltas and a terminal error. The configured request timeout covers the
entire stream, including callers that pass `context.Background()`. A clean provider terminal event or
`[DONE]` closes the channel; malformed, truncated, oversized, timed-out, and provider-failed streams
surface a terminal error.

Luas does not automatically retry generation. Retrying after a timeout or partial stream can create
duplicate work and cost, and a transport failure does not prove the provider did not execute the
request. A durable product workflow must define its own idempotency, attempt ledger, partial-output
policy, and provider-specific retry budget.

Streaming partial output is also harder to moderate than a completed response. Any downstream route
that exposes deltas to users owns moderation and policy enforcement before adopting this seam.

## Adding A Provider

1. Implement `Provider`; implement `StreamingProvider` only when the adapter has real stream support.
2. Add provider-specific typed secrets/endpoints to `AIConfig` and register the adapter in
   `NewManager`.
3. Extend startup validation and `doctor` checks. Do not read environment variables in the adapter.
4. Apply the same timeout, redirect, byte-limit, error privacy, and request-validation contract.
5. Add `httptest` coverage for success, provider status, oversized responses, redirects, timeout,
   malformed/truncated streams, and response-body privacy.
6. Update this document and the AI boundary governance script.

## Verification

```bash
cd api
go test ./internal/capabilities/ai ./internal/infra/config ./internal/infra/console/commands
go test -race ./internal/capabilities/ai
```

The adapter follows the official [Responses API reference](https://developers.openai.com/api/reference/resources/responses/methods/create)
and [streaming event guidance](https://developers.openai.com/api/docs/guides/streaming-responses).
