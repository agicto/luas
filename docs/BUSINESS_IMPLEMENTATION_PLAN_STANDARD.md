# Business Implementation Plan Standard

This document defines the minimum planning standard for a business capability
before implementation begins in Luas. It is a reusable specification, not an
implementation plan for a particular product.

The purpose of the plan is to turn a product requirement into a reviewable set
of contracts, ownership boundaries, invariants, delivery slices, and proof. A
plan is ready for implementation only when another engineer can identify what
must be built, where it belongs, how it behaves, and how completion will be
verified without reconstructing the design from mockups or conversations.

Normative words have their usual meaning:

- **MUST** is required before implementation starts.
- **SHOULD** is expected unless the plan records a concrete reason not to do it.
- **MAY** is optional.

## 1. When This Standard Applies

Use this standard for a new business capability, a material workflow, or a
change that crosses an HTTP contract or deployable-unit boundary.

A small local change MAY use a reduced plan when it preserves the public
contract, introduces no new state or authorization rule, and has an obvious
verification path. The reduced plan must still state scope, affected owner,
acceptance criteria, and verification.

The plan does not replace the authority of:

- `CONTEXT.md` for global vocabulary and surface ownership;
- `docs/ARCHITECTURE.md` for deployable-unit boundaries and vertical flow;
- the owning file under `contracts/` for public HTTP behavior;
- the nearest `AGENTS.md` for local implementation rules.

## 2. Planning Principles

Every implementation plan MUST follow these principles.

### 2.1 Describe behavior before code

Start with actors, outcomes, workflows, invariants, and failure behavior. File
trees and framework types support the design; they are not the design.

### 2.2 Assign one owner to each rule

Every business rule, persisted record, public endpoint, event, and scheduled
operation MUST have one owning module or deployable unit. Read models may
aggregate data from several owners, but they must not become a second writer.

### 2.3 Keep deployable units independent

`api/`, `web/`, and `admin/` communicate through reviewed HTTP behavior. They
MUST NOT import one another's source. The plan MUST name every deployable unit
that changes and the contract joining them.

### 2.4 Prefer vertical delivery slices

The implementation sequence SHOULD deliver an end-to-end usable workflow
rather than finishing all persistence, then all API work, then all UI work.
Each slice should include the contract, owning API behavior, relevant browser
surface, tests, and observable acceptance result.

### 2.5 Make derived records deterministic

Records produced from another operation, such as ledger entries, projections,
notifications, or generated schedules, MUST name their trigger, source of
truth, idempotency key, retry behavior, and reconciliation path.

### 2.6 Treat authorization and concurrency as business behavior

Role checks alone are not sufficient. The plan MUST cover resource ownership,
tenant scope when applicable, current state, conflicting writes, and what the
caller observes when a check fails.

## 3. Required Plan Structure

An implementation plan MUST contain the following sections. A section may be
marked not applicable only with a short reason.

## 3.1 Executive Summary

State:

- the capability and user outcome;
- the problem being solved;
- the in-scope workflow;
- explicit non-goals;
- affected deployable units;
- the success signal for the first release.

Do not list pages or endpoints without explaining the outcome they support.

## 3.2 Actors and Access Surfaces

For each actor, record:

| Field | Required content |
|---|---|
| Actor | Stable domain name, not a screen label |
| Goal | What the actor is trying to achieve |
| Entry points | Routes, commands, jobs, or external calls |
| Read scope | Which records the actor may discover |
| Mutation scope | Which records and transitions the actor may change |
| Approval boundary | Whether another actor or system must approve |
| Denial behavior | Status and stable error class without information leakage |

If one actor inherits another actor's abilities, the plan MUST still list any
additional restrictions and separation-of-duty rules.

## 3.3 End-to-End Workflows

Describe each primary workflow from initiation to terminal outcome. Include:

1. initiating actor and preconditions;
2. authoritative input and validation;
3. records created or changed;
4. state transitions;
5. cross-module calls or emitted events;
6. user-visible result;
7. rejection and recovery paths;
8. timeout, cancellation, and retry behavior;
9. audit or notification requirements.

Use a sequence or state diagram when three or more components participate.
Happy-path prose alone is insufficient.

## 3.4 Module Decomposition

The plan MUST include a module catalog:

| Field | Meaning |
|---|---|
| Module | Domain concept owned by the module |
| Responsibility | Rules and lifecycle it owns |
| Inputs | Commands, requests, events, or jobs it accepts |
| Outputs | Responses, events, or records it produces |
| Data ownership | Tables or durable state for which it is the sole writer |
| Dependencies | Required modules or infrastructure capabilities |
| Consumers | Browser surfaces, modules, jobs, or integrations |
| Priority | Delivery priority justified by dependency or risk |

Module names MUST describe domain concepts rather than transport actions. A
technical adapter that owns no route or business lifecycle belongs under an
infrastructure capability, not a business module.

For every module, specify:

- responsibilities and explicit exclusions;
- public and internal entry points;
- invariants enforced by the service layer;
- data it owns and data it only reads;
- synchronous dependencies and event dependencies;
- authorization and resource-scope rules;
- concurrency, idempotency, and transaction boundaries;
- failure modes and recovery;
- module-level acceptance criteria.

## 3.5 Domain Model and Vocabulary

Define the stable business terms before naming code. For each aggregate or
entity, record:

- identity and lifecycle;
- owning module;
- mutable and immutable attributes;
- value objects and units;
- invariants;
- relationships and cardinality;
- source-of-truth fields versus snapshots or derived fields;
- retention, archival, and deletion behavior.

Avoid using one field for independent concepts. For example, fulfillment,
approval, and payment lifecycles require separate states when they can change
independently.

## 3.6 Persistence Design

For each new or changed table, the plan MUST specify:

- purpose and owning module;
- columns, types, nullability, defaults, and units;
- primary, unique, foreign-key, and query indexes;
- check constraints where the database can protect an invariant;
- audit timestamps and soft-delete behavior when applicable;
- snapshot and derived columns;
- optimistic version or locking strategy where concurrent mutation is possible;
- data retention and cleanup behavior;
- forward migration, rollback strategy, and backfill;
- seed data and whether it is environment-independent.

Additional Luas requirements:

- PostgreSQL is the only relational compatibility target.
- Migrations and seeders own schema and initial data changes; plans MUST NOT
  require manual production DDL or data edits.
- Monetary values MUST use an exact representation with a declared currency
  and rounding rule. Floating-point values are not acceptable for money.
- Time fields MUST declare timezone semantics. Recurring local-time rules must
  state how daylight-saving and timezone changes are handled.
- History that must survive later edits SHOULD be stored as an explicit
  snapshot rather than reconstructed from mutable records.

## 3.7 State Machines

Every entity with a lifecycle MUST have a state definition and transition
table:

| Current state | Command or trigger | Guard | Next state | Actor | Side effects | Failure |
|---|---|---|---|---|---|---|

The state-machine section MUST cover:

- initial and terminal states;
- all legal transitions;
- rejected transitions and stable error codes;
- restoration or compensation after a rejected approval;
- timeout and automatic transitions;
- repeated commands and idempotent results;
- concurrent transitions and conflict behavior;
- whether side effects occur before, within, or after the state commit.

Do not rely on a generic update endpoint to mutate lifecycle state. Important
transitions SHOULD have explicit commands or subresources.

## 3.8 HTTP Contract Design

Public behavior MUST be documented under `contracts/` before multiple
deployable units implement it. For each endpoint, specify:

1. method and path;
2. owning module and intended consumers;
3. authentication and authorization;
4. path, query, header, and body fields;
5. validation rules and defaults;
6. service-layer operation;
7. transaction and idempotency behavior;
8. success response and pagination when applicable;
9. stable non-2xx `error_code` values;
10. rate limit, size limit, and timeout where relevant;
11. audit and observability requirements;
12. compatibility and rollout notes.

All API responses MUST follow the shared envelope in `contracts/README.md`.
Browser behavior MUST branch on `error_code`, not backend message text.

List endpoints MUST define:

- pagination or an explicit bounded non-paginated result;
- filter and sort semantics;
- maximum page or date-range limits;
- visibility scope;
- deterministic ordering.

Upload and download endpoints MUST additionally define file type, size,
inspection state, storage ownership, access duration, and deletion semantics.

### 3.8.1 Interface Pseudocode Standard

Every material command endpoint MUST include behavioral pseudocode. Simple
bounded queries MAY use a query specification instead, but a sentence such as
"query the records and return them" is not sufficient when visibility,
aggregation, or derived fields are involved.

Pseudocode exists to make domain behavior reviewable before implementation. It
MUST describe observable decisions and atomic work without pretending to be
compilable Go or embedding a full repository implementation.

#### Required execution phases

Write command pseudocode in the following order. Omit a phase only when it is
not applicable.

1. **Bind** — obtain the authenticated context and bind path, query, header,
   and body inputs.
2. **Validate input** — validate shape, enum membership, ranges, formats,
   conditional requirements, and request-size limits.
3. **Authorize scope** — establish the caller's capability and authoritative
   resource scope before revealing protected existence.
4. **Load authoritative state** — load the aggregate and required configuration
   or policy inputs from their owners.
5. **Check preconditions** — enforce current state, ownership, uniqueness,
   deadlines, quotas, and cross-field invariants.
6. **Derive values** — calculate identifiers, amounts, snapshots, deadlines,
   next state, and idempotency keys from named sources.
7. **Commit atomic changes** — declare the transaction owner and list every
   write that must succeed or fail together.
8. **Record durable side effects** — write an outbox/work item in the same
   transaction when work continues asynchronously.
9. **Run post-commit work** — perform only explicitly safe external work after
   commit; state its retry and reconciliation behavior.
10. **Build response** — map the committed domain result to the documented
    response without exposing persistence-only fields.

Validation order is part of the behavior when it affects disclosure or error
precedence. Authorization and scoped lookup SHOULD normally precede a detailed
state error so an unauthorized caller cannot discover a protected resource.

#### Normative pseudocode vocabulary

Use the following uppercase operations consistently:

| Operation | Meaning |
|---|---|
| `REQUIRE condition ELSE error_code` | Reject unless a transport or domain condition holds |
| `LOAD name FROM owner USING key` | Read authoritative state from its owning boundary |
| `LOAD_FOR_UPDATE` | Read with an explicitly justified database lock |
| `DERIVE name = expression` | Compute a value without side effects |
| `BEGIN TRANSACTION owner` | Start the named atomic persistence boundary |
| `INSERT` / `UPDATE` / `DELETE` | Mutate owner-controlled persistence |
| `CAS_UPDATE ... EXPECT version` | Perform an optimistic compare-and-swap update |
| `ENQUEUE_OUTBOX event KEY key` | Durably record an asynchronous event or command |
| `COMMIT` | Make all transaction writes visible atomically |
| `AFTER COMMIT` | Begin work that is intentionally outside the transaction |
| `RETURN status response` | Return the documented public result |

Names in pseudocode SHOULD refer to domain objects and owning operations. Raw
SQL MAY be included only when a database-specific constraint, lock, aggregate,
or compare-and-swap statement is essential to the design. Do not couple every
service step to a table name merely to make the plan look concrete.

#### Command pseudocode template

```text
COMMAND <operation>(actor, path, request, idempotency_key?)

BIND actor FROM authenticated_context
VALIDATE request AGAINST <request schema and limits>
REQUIRE actor MAY <capability> IN <scope> ELSE <AUTH.* error_code>

LOAD aggregate FROM <owning module> USING path.id
  OR ELSE <not-found/non-disclosure error_code>
LOAD policy FROM <owning module> USING <policy key>

REQUIRE aggregate.state IN {<allowed states>}
  ELSE <MODULE.INVALID_STATE>
REQUIRE <ownership/deadline/quota/uniqueness condition>
  ELSE <stable error_code>

DERIVE next_state = <transition rule>
DERIVE snapshot = <authoritative source fields>
DERIVE operation_key = <stable uniqueness/idempotency source>

BEGIN TRANSACTION <transaction owner>
  REQUIRE operation_key HAS NOT BEEN COMMITTED
    ELSE RETURN <documented replay result or conflict>
  CAS_UPDATE aggregate EXPECT aggregate.version
    SET state = next_state, version = version + 1
    ELSE <MODULE.CONCURRENT_MODIFICATION>
  INSERT dependent_record WITH snapshot, operation_key
  ENQUEUE_OUTBOX <event name and version> KEY operation_key
COMMIT

AFTER COMMIT
  <worker/provider action, retry policy, reconciliation owner>

RETURN <HTTP status> <response DTO built from committed result>
```

The plan MUST replace every placeholder with a decided rule. Phrases such as
"if needed," "optionally," "recommended," or "depending on requirements"
are unresolved design decisions and are not valid inside implementation-ready
pseudocode.

#### Query specification template

Queries SHOULD be described as a deterministic read pipeline:

```text
QUERY <operation>(actor, filters, page)

BIND actor FROM authenticated_context
VALIDATE filters, sort, page, and maximum range
REQUIRE actor MAY <read capability> IN <scope> ELSE <AUTH.* error_code>

DERIVE visibility_predicate FROM verified actor scope
LOAD page FROM <read owner>
  WHERE visibility_predicate
    AND <filter semantics>
  ORDER BY <stable field> <direction>, id <direction>
  LIMIT <bounded page size>

DERIVE response_fields FROM <authoritative/snapshot/aggregate sources>
RETURN 200 <paginated response DTO>
```

For an intentionally non-paginated query, state the finite catalog or maximum
date range that bounds the result. Aggregate queries MUST define the formula,
included states, time boundary, timezone, treatment of empty input, and whether
results are live, cached, or precomputed.

#### Branches and state transitions

Use explicit branches when an operation has materially different outcomes:

```text
IF decision == approved
  REQUIRE <approval guards> ELSE <stable error_code>
  <approved transition and side effects>
ELSE IF decision == rejected
  REQUIRE rejection_reason IS PRESENT ELSE <validation error_code>
  <rejected transition and side effects>
ELSE
  RETURN <validation error_code>
```

Do not hide separate domain commands behind a generic `PATCH status` when the
branches have different permissions, invariants, side effects, or audit needs.

#### Transaction and side-effect rules

Pseudocode MUST make atomicity visually unambiguous:

- Indent every atomic read/write between `BEGIN TRANSACTION` and `COMMIT`.
- Name the transaction owner.
- State the lock order when more than one aggregate is locked.
- Use `CAS_UPDATE` or `LOAD_FOR_UPDATE` only with a stated concurrency reason.
- Never place an object-store call, email, webhook, or other remote provider
  call inside a database transaction unless the architecture explicitly owns a
  compensating protocol.
- If an event must not be lost, use a durable outbox or perform the derived
  local write in the same transaction. `COMMIT` followed by an untracked
  `Publish()` is not sufficient.
- Event pseudocode must include event name, schema version, aggregate ID,
  causation/correlation information when applicable, and idempotency key.

#### Error mapping

Each pseudocode guard MUST map to a stable public error. Add an error table
next to the algorithm:

| Condition | HTTP status | `error_code` | Disclosure rule | Retryable |
|---|---:|---|---|---|
| Malformed transport input | 400 | `COMMON.INVALID_INPUT` | Field-neutral | No |
| Schema or field violation | 422 | `COMMON.VALIDATION_FAILED` | Reviewed field keys only | After correction |
| Missing authentication | 401 | `AUTH.UNAUTHORIZED` | No resource detail | After authentication |
| Forbidden scoped access | 403 or 404 | `<documented code>` | Follow non-disclosure policy | No |
| State or uniqueness conflict | 409 | `<MODULE.CONFLICT>` | Safe domain detail only | Possibly |
| Concurrent modification | 409 | `<MODULE.CONCURRENT_MODIFICATION>` | Prompt refetch | Yes |
| Dependency unavailable | 503 | `COMMON.SERVICE_UNAVAILABLE` | No provider detail | Yes |

The plan MUST use `error_code` as the contract. Human-readable messages are
illustrative copy only and cannot be the sole failure definition.

#### Response pseudocode

Success behavior MUST state:

- HTTP status;
- response DTO or schema reference;
- which values are authoritative, snapshotted, derived, or redacted;
- whether the returned state is committed and immediately readable;
- pagination metadata for lists;
- relevant headers such as `Location`, `ETag`, or download metadata.

Do not use `complete object`, `same as above`, or an ellipsis as the only
response definition in an implementation-ready plan.

#### Pseudocode review checklist

- [ ] Steps follow observable execution order.
- [ ] Inputs, defaults, conditional requirements, and limits are explicit.
- [ ] Authorization includes resource scope and non-disclosure behavior.
- [ ] Every load names an authoritative owner.
- [ ] Every guard maps to a stable error code.
- [ ] Calculations name units, precision, timezone, and rounding where relevant.
- [ ] State transition matches the state-machine table.
- [ ] Transaction start, writes, durable events, and commit are visibly bounded.
- [ ] Concurrency and duplicate requests have deterministic outcomes.
- [ ] External side effects have retry and reconciliation behavior.
- [ ] Response status and DTO are explicit.
- [ ] No unresolved alternatives remain in the algorithm.

## 3.9 Authorization Matrix

Include a matrix mapping actors to capabilities. Each cell MUST distinguish at
least:

- forbidden;
- own resources only;
- scoped resources, such as an active organization;
- all resources within an administrative boundary;
- approval-only access.

Below the matrix, document row-level constraints, non-disclosure behavior,
separation of duties, and any operation an actor cannot perform on its own
record.

Client-side route guards are user experience controls, not authorization. The
API remains authoritative.

## 3.10 Cross-Module Data Flow

For every workflow involving more than one module, identify:

- transaction owner;
- synchronous participants;
- event producer and consumers;
- committed source of truth;
- event or command payload contract;
- idempotency key;
- delivery guarantee and retry policy;
- ordering requirement;
- duplicate and out-of-order handling;
- dead-letter or failed-work visibility;
- reconciliation or replay mechanism.

Use one database transaction only for invariants within a supported local
boundary. Work that crosses deployable services or background jobs requires an
explicit event, outbox, or workflow contract.

The plan MUST include a derived-record table:

| Trigger | Source record | Generated record/action | Owner | Idempotency key | Failure recovery |
|---|---|---|---|---|---|

## 3.11 Scheduled and Asynchronous Work

For each timeout, reminder, recurring generation, export, or cleanup job,
record:

- trigger or schedule;
- timezone;
- selection query and batch bound;
- lease or concurrency control;
- idempotency and retry;
- maximum attempts and backoff;
- failure visibility;
- manual replay or reconciliation;
- retention and cleanup.

A timestamp alone does not implement a timeout. The plan must name the process
that observes it and performs the transition.

## 3.12 Browser-Surface Plan

For every changed `web/` or `admin/` surface, specify:

- route and owning feature;
- contract or fixed browser adapter used;
- query, mutation, and cache invalidation behavior;
- loading, empty, success, validation, conflict, forbidden, and unavailable states;
- filtering, pagination, and URL state;
- responsive and accessibility behavior;
- role-based navigation behavior;
- localization keys and user-facing error mapping;
- upload, download, and export interaction where applicable;
- development mock parity and production replacement boundary.

For Web, server-owned authenticated adapters remain allowlisted and explicit.
For Admin, the plan must respect that the application is static and cannot own
private server runtime behavior.

## 3.13 Security, Privacy, and Audit

The plan MUST identify:

- sensitive and personal data;
- who may read or mutate it;
- data exposed in lists, logs, events, exports, and notifications;
- encryption or secret-storage requirements;
- upload inspection and content access rules;
- audit events for privileged or financially significant operations;
- retention and deletion requirements;
- abuse limits, rate limits, and enumeration risks;
- CSRF, Origin, session, and browser-gateway requirements for protected UI.

Do not place secrets, provider responses, internal storage keys, or unnecessary
personal data in public DTOs or event payloads.

## 3.14 Observability and Operations

For material workflows, define:

- structured log events and privacy-safe fields;
- counters, latency, failure, backlog, and retry metrics;
- tracing boundaries and correlation identifiers;
- alerts and actionable thresholds;
- operational status or reconciliation views;
- support information, including `request_id` propagation;
- rollback, disable switch, or safe-degradation behavior.

## 3.15 Test and Verification Strategy

Map each important invariant to proof at the narrowest useful layer:

| Concern | Required proof |
|---|---|
| Pure calculation or transition | Unit test |
| Service invariant | Service test with existing seams or doubles |
| PostgreSQL constraint/query/locking | Disposable PostgreSQL test |
| HTTP validation and authorization | Handler or integration test |
| Public contract | Contract and browser-consumer tests |
| Browser workflow | Hook/component/route test |
| Cross-module event | Producer, consumer, and idempotency test |
| Concurrency conflict | Deterministic concurrent or lock test |
| Migration/backfill | Clean install and upgrade-path proof |

The plan MUST list critical scenarios, including:

- happy paths and terminal outcomes;
- invalid state transitions;
- role and resource-scope violations;
- duplicate requests and duplicate events;
- concurrent mutation;
- boundary values, timeouts, and clock edges;
- partial dependency failure and retry;
- pagination, filtering, and stable ordering;
- migration compatibility and rollback.

The final verification commands must follow the nearest `AGENTS.md` matrix.
Do not introduce SQLite fixtures or dependencies for relational behavior.

## 3.16 Delivery Plan

Divide delivery into dependency-aware vertical slices. Each slice MUST contain:

- outcome and included workflow;
- modules and deployable units changed;
- contract and migration changes;
- feature flag, configuration, or rollout dependency;
- tests and acceptance evidence;
- data migration or backfill;
- rollback or disable path;
- explicit exclusions deferred to a later slice.

Estimates SHOULD state assumptions and risk rather than presenting an
unsupported calendar number. Identify the critical path and work that can run
in parallel.

## 4. Module Detail Template

Use this template for every planned module:

```markdown
## Module: <domain concept>

### Objective
<User or system outcome owned by this module.>

### Responsibilities
- ...

### Explicit exclusions
- ...

### Actors and access
- <actor>: <read/mutation scope>

### Domain objects and invariants
- <object>: <identity, lifecycle, invariants>

### Owned persistence
- <table/record>: <purpose, keys, constraints, retention>

### Public contract
- <method path>: <command/query, success, stable errors>

### Internal dependencies
- Synchronous: ...
- Events consumed: ...
- Events published: ...
- Infrastructure capabilities: ...

### Transactions, concurrency, and idempotency
- ...

### Failure and recovery
- ...

### Browser surfaces
- ...

### Observability and audit
- ...

### Acceptance criteria
- Given ..., when ..., then ...

### Verification
- <focused command or test>
```

## 5. Endpoint Detail Template

````markdown
### <METHOD> <PATH> — <business operation>

- Owner:
- Consumers:
- Authentication:
- Authorization and resource scope:
- Request:
- Validation and limits:
- Preconditions:
- Pseudocode:
  ```text
  COMMAND or QUERY <operation>(...)
  BIND ...
  VALIDATE ...
  REQUIRE ... ELSE <error_code>
  LOAD ...
  DERIVE ...
  BEGIN TRANSACTION <owner>
    ...
  COMMIT
  AFTER COMMIT ...
  RETURN <status> <response DTO>
  ```
- Transaction boundary:
- Idempotency/concurrency:
- Side effects/events:
- Success response:
- Stable errors:

  | Condition | HTTP status | `error_code` | Retryable |
  |---|---:|---|---|
  | ... | ... | ... | ... |

- Audit/metrics:
- Compatibility notes:
- Tests:
````

## 6. Pre-Implementation Readiness Gate

Implementation MUST NOT begin until all blocking items below are resolved.

For a durable plan, run the mechanical lint before the semantic review:

```bash
python3 .agents/skills/business-implementation-planning/scripts/validate_plan.py --strict <plan-path>
```

The script detects structural omissions, unresolved placeholders, unbalanced
code fences, and likely command/pseudocode drift. Passing it does not prove the
domain decisions are correct and does not replace the checklist below.

### Product and domain

- [ ] Outcome, scope, non-goals, and actors are explicit.
- [ ] Vocabulary is consistent with `CONTEXT.md`.
- [ ] Primary, rejection, cancellation, timeout, and recovery flows are defined.
- [ ] State machines contain no missing or contradictory transitions.
- [ ] Independent lifecycles are modeled independently.
- [ ] Amount, quantity, time, and timezone semantics are unambiguous.

### Ownership and architecture

- [ ] Each rule, record, route, event, and job has one owner.
- [ ] Module responsibilities and exclusions are explicit.
- [ ] Deployable units communicate only through contracts.
- [ ] Synchronous and asynchronous dependencies are listed.
- [ ] Transaction boundaries do not cross unsupported runtime boundaries.

### Data and contract

- [ ] Schema, constraints, indexes, migration, rollback, and backfill are defined.
- [ ] Public HTTP behavior is documented under `contracts/`.
- [ ] Authorization includes row/tenant scope, not roles alone.
- [ ] Pagination, filtering, ordering, limits, and errors are defined.
- [ ] Idempotency and concurrency behavior are defined for every material mutation.

### Side effects and operations

- [ ] Derived records have one trigger and an idempotency key.
- [ ] Events define delivery, retry, duplicate, ordering, and replay behavior.
- [ ] Scheduled work names an executable owner and failure visibility.
- [ ] Audit, privacy, metrics, alerts, and support diagnostics are defined.
- [ ] Rollout, rollback, and safe disable behavior are defined.

### Delivery and proof

- [ ] Vertical slices are dependency ordered.
- [ ] Each slice has observable acceptance criteria.
- [ ] Critical invariants map to concrete tests.
- [ ] PostgreSQL-specific behavior has PostgreSQL proof.
- [ ] Verification commands match the affected deployable units.

## 7. Common Planning Defects

A plan is not implementation-ready when it has any of these defects:

- a workflow references a state or field absent from the model;
- a rejection path says "restore" without persisting the prior state;
- a deadline exists without a worker or request path that enforces it;
- one event can generate the same financial or derived record twice;
- monetary values use floating point or omit currency and rounding;
- an aggregate is updated by more than one module without an ownership rule;
- a UI route guard is treated as authorization;
- a list endpoint has no result bound or deterministic ordering;
- an upload stores a durable public URL without a private access model;
- an asynchronous side effect has no retry, reconciliation, or failure view;
- an estimate is given before dependencies, migration, and acceptance scope are known;
- database behavior is claimed from a compatibility target Luas does not support.

## 8. Definition of Ready

A business implementation plan is ready when:

1. the readiness gate is complete;
2. no unresolved decision can materially change the data model, public
   contract, authorization boundary, state machine, or delivery sequence;
3. the first vertical slice can be implemented and verified from the plan;
4. reviewers can trace every user-visible outcome to an owning module,
   persisted state, contract, and test.

Open questions that do not block the first slice MAY remain only when their
scope, owner, decision deadline, and containment strategy are recorded.
