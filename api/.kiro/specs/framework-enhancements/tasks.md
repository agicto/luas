# Implementation Plan: Framework Enhancements

## Overview

This implementation plan covers framework enhancements in two main areas: Event System and Testing Tools.

**Module System (Tasks 6-7) has been removed** - see rationale below.

## Reference Framework Analysis Summary

Based on deep analysis of go-zero, Kratos, go-kit, and other enterprise Go projects:

### What ZGO Already Has (Well Implemented)
- **Event System** (`internal/infra/events/`) - EventBus with priority, wildcard, middleware ✅
- **Lifecycle** (`internal/infra/lifecycle/`) - Hook-based start/stop with rollback ✅
- **Circuit Breaker** (`internal/infra/breaker/`) - Similar to go-zero's implementation ✅
- **Rate Limiting** (`internal/infra/ratelimit/`) - In-memory with Gin middleware ✅
- **Singleflight** (`internal/infra/singleflight/`) - Request deduplication ✅
- **Health Checks** (`internal/infra/health/`) - Health check system ✅
- **Structured Errors** (`pkg/errors/`) - Error codes with HTTP status mapping ✅

### Potential Future Enhancements (Not in Current Scope)
- **Redis-based Rate Limiting** - go-zero has Redis + fallback pattern for distributed deployments
- **gRPC Status Integration** - Kratos pattern, only needed if gRPC support is planned

## Current Implementation Status

Completed:
- `internal/infra/events/types.go` - Event interface, metadata, BaseEvent ✅
- `internal/infra/events/bus.go` - EventBus with priority, wildcard, middleware ✅
- `internal/infra/events/pattern.go` - Glob-style pattern matching ✅
- `internal/infra/events/example_test.go` - Usage examples ✅
- `internal/infra/lifecycle/lifecycle.go` - Lifecycle management ✅

Pending:
- Event middleware implementations (logging, tracing, recovery)
- Event persistence (optional)
- Testing tools (Suite Builder, Fixtures, Mock Builder, Assertions)

## Tasks

- [x] 1. Event System Core Enhancement
  - [x] 1.1 Enhance event types with metadata
    - Create `internal/infra/events/types.go`
    - Define `EventMetadata` struct with ID, CorrelationID, CausationID, Timestamp, Source
    - Update `Event` interface to include `Metadata() EventMetadata`
    - Implement `BaseEvent` with automatic metadata generation (UUID for event_id)
    - _Requirements: 1.2_

  - [ ]* 1.2 Write property test for event metadata
    - **Property 2: Event Metadata Presence**
    - **Validates: Requirements 1.2**

  - [x] 1.3 Implement enhanced EventBus
    - Create `internal/infra/events/bus.go`
    - Migrate from existing `internal/infra/event/event.go`
    - Add `Subscribe()` with `SubscribeOption` for priority
    - Add `Unsubscribe()` method returning `Subscription` interface
    - Implement handler registration with priority support (higher first)
    - Support context cancellation during dispatch
    - _Requirements: 1.1, 1.3, 2.1, 2.2, 2.3, 2.5_

  - [ ]* 1.4 Write property test for event delivery
    - **Property 1: Event Delivery Completeness**
    - **Validates: Requirements 1.1, 2.1, 2.2**

  - [ ]* 1.5 Write property test for priority ordering
    - **Property 4: Priority-Based Handler Ordering**
    - **Validates: Requirements 2.3**

  - [x] 1.6 Implement wildcard subscription matching
    - Add glob-style pattern matching for event names
    - Support patterns like "user.*", "*.created", "*"
    - Use `path.Match` or custom implementation
    - _Requirements: 2.4_

  - [ ]* 1.7 Write property test for wildcard matching
    - **Property 5: Wildcard Pattern Matching**
    - **Validates: Requirements 2.4**

  - [ ]* 1.8 Write property test for unsubscription
    - **Property 6: Unsubscription Effectiveness**
    - **Validates: Requirements 2.5**

- [ ] 2. Checkpoint - Event System Core
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 3. Event Middleware
  - [ ] 3.1 Create middleware interface and chain
    - Create `internal/infra/events/middleware.go`
    - Define `EventMiddleware` type as `func(next EventHandler) EventHandler`
    - Implement middleware chain execution in EventBus
    - _Requirements: 3.1, 3.5_

  - [ ]* 3.2 Write property test for middleware ordering
    - **Property 7: Middleware Chain Ordering**
    - **Validates: Requirements 3.1, 3.5**

  - [ ] 3.3 Implement logging middleware
    - Create `LoggingMiddleware(logger Logger) EventMiddleware`
    - Support configurable log levels (debug, info, warn, error)
    - Log event name, duration, and errors
    - _Requirements: 3.2_

  - [ ] 3.4 Implement tracing middleware
    - Create `TracingMiddleware(tracer trace.Tracer) EventMiddleware`
    - Integrate with existing `internal/infra/tracing/`
    - Create spans for event handling with event metadata as attributes
    - _Requirements: 3.3_

  - [ ] 3.5 Implement recovery middleware
    - Create `RecoveryMiddleware(onPanic func(error)) EventMiddleware`
    - Handle panics in handlers, convert to errors
    - Create `RetryMiddleware(maxRetries int, backoff time.Duration) EventMiddleware`
    - _Requirements: 3.4_

  - [ ]* 3.6 Write property test for retry behavior
    - **Property 8: Retry Middleware Behavior**
    - **Validates: Requirements 3.4**

  - [ ]* 3.7 Write property test for error propagation
    - **Property 3: Synchronous Error Propagation**
    - **Validates: Requirements 1.4**

- [ ] 4. Event Persistence (Optional)
  - [ ] 4.1 Create event store interface and model
    - Create `internal/infra/events/store.go`
    - Define `EventStore` interface with `Store()`, `Query()`, `Replay()`
    - Create `StoredEvent` GORM model with indexes
    - _Requirements: 4.1_

  - [ ] 4.2 Implement GORM-based event store
    - Implement `GormEventStore` struct
    - Support filtering by event type, time range, correlation_id
    - Implement chronological replay
    - _Requirements: 4.1, 4.2, 4.3_

  - [ ]* 4.3 Write property test for event persistence round-trip
    - **Property 9: Event Persistence Round-Trip**
    - **Validates: Requirements 4.1, 4.2**

  - [ ]* 4.4 Write property test for replay ordering
    - **Property 10: Event Replay Chronological Order**
    - **Validates: Requirements 4.3**

- [ ] 5. Checkpoint - Event System Complete
  - Ensure all tests pass, ask the user if questions arise.

## ~~Module System (Tasks 6-7) - REMOVED~~

> **Decision**: After deep analysis of Uber Fx and comparison with ZGO's current architecture,
> the module system tasks (6.1-6.5, 7.1-7.6) have been **removed** from this plan.
>
> **Rationale**:
> - ZGO already uses Wire + ProviderSet for modularization (see `internal/modules/*/provider.go`)
> - Go's package system provides natural module boundaries
> - Lifecycle management is already implemented in `internal/infra/lifecycle/`
> - Adding Fx-style module system would add complexity without significant benefit
> - ZGO is a single-app framework, not a plugin system requiring dynamic module loading
>
> **What ZGO Already Has**:
> - `ProviderSet` in each module for DI grouping
> - Central aggregation in `internal/wiring/wire.go`
> - Route registration via `RegisterRoutes()` methods
> - Lifecycle hooks for startup/shutdown

- [x] 6. Module System - SKIPPED (Wire + ProviderSet sufficient)
- [x] 7. Module Lifecycle - SKIPPED (Already implemented in lifecycle package)
- [x] 8. Checkpoint - Module System - SKIPPED

- [ ] 9. Testing Suite Builder
  - [ ] 9.1 Create test suite structure
    - Create `internal/infra/testing/suite.go`
    - Implement `Suite` struct with `NewSuite(t *testing.T)`
    - Implement `Describe()`, `It()` for test organization
    - Implement `Skip()`, `Focus()` for test filtering
    - _Requirements: 9.1, 9.3_

  - [ ] 9.2 Implement test hooks
    - Implement `BeforeEach()`, `AfterEach()` (per test)
    - Implement `BeforeAll()`, `AfterAll()` (per suite)
    - Ensure proper execution order
    - _Requirements: 9.2_

  - [ ]* 9.3 Write property test for hook execution order
    - **Property 19: Test Hook Execution Order**
    - **Validates: Requirements 9.2**

  - [ ] 9.4 Implement table-driven test builder
    - Implement `TestCases()` builder method
    - Support parameterized tests with `TestCase` struct
    - _Requirements: 9.4_

- [ ] 10. Test Fixtures
  - [ ] 10.1 Create fixture base
    - Create `internal/infra/testing/fixture.go`
    - Implement `Fixture` struct with options pattern
    - Implement cleanup registration (LIFO order)
    - _Requirements: 10.3, 10.4_

  - [ ]* 10.2 Write property test for fixture cleanup
    - **Property 20: Fixture Cleanup**
    - **Validates: Requirements 10.3**

  - [ ] 10.3 Implement database fixture
    - Enhance existing `DatabaseTestCase`
    - Support in-memory SQLite with `WithDatabase()` option
    - Support migrations with `WithMigrations()` option
    - Support data seeding with `Seed()` method
    - _Requirements: 10.1, 14.2_

  - [ ] 10.4 Implement HTTP fixture
    - Integrate with existing `TestCase`
    - Support test router setup with `WithRouter()` option
    - _Requirements: 10.2_

  - [ ]* 10.5 Write property test for data isolation
    - **Property 21: Test Data Isolation**
    - **Validates: Requirements 10.5, 14.4**

  - [ ] 10.6 Implement transaction rollback support
    - Wrap tests in transactions
    - Rollback after test completion
    - _Requirements: 14.5_

  - [ ]* 10.7 Write property test for transaction rollback
    - **Property 31: Transaction Rollback**
    - **Validates: Requirements 14.5**

- [ ] 11. Mock Builder
  - [ ] 11.1 Create mock builder structure
    - Create `internal/infra/testing/mock.go`
    - Implement generic `Mock[T any]` struct
    - Implement `On()` method returning `Expectation`
    - Implement `Returns()` method
    - _Requirements: 11.1, 11.2_

  - [ ]* 11.2 Write property test for mock return values
    - **Property 22: Mock Return Values**
    - **Validates: Requirements 11.2**

  - [ ] 11.3 Implement call verification
    - Implement `Times(n)`, `Once()`, `Never()` methods
    - Implement `Verify(t *testing.T)` method
    - _Requirements: 11.3_

  - [ ]* 11.4 Write property test for call count verification
    - **Property 23: Mock Call Count Verification**
    - **Validates: Requirements 11.3**

  - [ ] 11.5 Implement argument matching
    - Implement `WithArgs()` matcher
    - Support `Any()` matcher for any argument
    - _Requirements: 11.4_

  - [ ]* 11.6 Write property test for argument matching
    - **Property 24: Mock Argument Matching**
    - **Validates: Requirements 11.4**

  - [ ] 11.7 Implement argument capture
    - Implement `Capture(ptr interface{})` method
    - Store captured arguments for later inspection
    - _Requirements: 11.5_

  - [ ]* 11.8 Write property test for argument capture
    - **Property 25: Mock Argument Capture**
    - **Validates: Requirements 11.5**

- [ ] 12. Assertion Helpers
  - [ ] 12.1 Create assertion base
    - Create `internal/infra/testing/assert.go`
    - Implement `Assert` struct with `NewAssert(t *testing.T)`
    - Implement `Equal()` with diff output using `go-cmp`
    - _Requirements: 12.5_

  - [ ] 12.2 Implement JSON assertions
    - Implement `JSONPath(json []byte, path string)` assertion
    - Support nested path extraction (dot notation)
    - Enhance existing `AssertJSONPath` in `TestResponse`
    - _Requirements: 12.1_

  - [ ]* 12.3 Write property test for JSON path assertion
    - **Property 26: JSON Path Assertion**
    - **Validates: Requirements 12.1**

  - [ ] 12.4 Implement database assertions
    - Enhance existing `DatabaseTestCase`
    - Ensure `DatabaseHas()`, `DatabaseMissing()`, `DatabaseCount()` work with `Assert`
    - _Requirements: 12.2_

  - [ ]* 12.5 Write property test for database assertions
    - **Property 27: Database State Assertion**
    - **Validates: Requirements 12.2**

  - [ ] 12.6 Implement time assertions
    - Implement `TimeEquals(actual, expected time.Time, tolerance time.Duration)`
    - Implement `TimeAfter()`, `TimeBefore()`
    - _Requirements: 12.4_

  - [ ]* 12.7 Write property test for time assertions
    - **Property 28: Time Assertion with Tolerance**
    - **Validates: Requirements 12.4**

- [ ] 13. Event Testing
  - [ ] 13.1 Create event recorder
    - Create `internal/infra/testing/events.go`
    - Implement `EventRecorder` struct for capturing events
    - Implement `Record()`, `Events()`, `EventsNamed()`, `Clear()`
    - _Requirements: 13.1_

  - [ ]* 13.2 Write property test for event capture
    - **Property 29: Event Capture Completeness**
    - **Validates: Requirements 13.1, 13.2, 13.3**

  - [ ] 13.3 Implement event assertions
    - Implement `EventAssertion` struct
    - Implement `EventPublished()`, `EventNotPublished()`
    - Implement `WithPayload()` assertion
    - _Requirements: 13.2, 13.3_

  - [ ] 13.4 Implement event order assertions
    - Implement `EventsInOrder()` assertion
    - Implement `Before()`, `After()` assertions
    - _Requirements: 13.4_

  - [ ]* 13.5 Write property test for event order
    - **Property 30: Event Order Assertion**
    - **Validates: Requirements 13.4**

- [ ] 14. Checkpoint - Testing Utilities Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 15. Integration and Documentation
  - [ ] 15.1 Integrate event system with existing domain events
    - Update `internal/domain/events.go` to use new `EventMetadata`
    - Migrate existing event types to include metadata
    - Update existing event handlers to use new EventBus
    - _Requirements: 1.1, 1.2, 2.1_

  - [ ] 15.2 Create module examples
    - Create example module using new module system
    - Document module creation pattern in `docs/guide/`
    - _Requirements: 5.1, 6.1_

  - [ ] 15.3 Update test setup to use new fixtures
    - Update `tests/feature/setup.go` to use new fixtures
    - Migrate existing tests to new patterns where beneficial
    - _Requirements: 10.1, 10.2_

  - [ ] 15.4 Add Wire providers for new components
    - Create provider sets for EventBus, ModuleRegistry
    - Update `internal/infra/provider.go` with new providers
    - _Requirements: 8.3_

- [ ] 16. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- The implementation order ensures dependencies are available when needed
- Existing implementations in `internal/infra/event/` and `internal/infra/testing/` should be enhanced, not replaced
