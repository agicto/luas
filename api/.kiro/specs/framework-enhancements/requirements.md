# Requirements Document

## Introduction

本文档定义了 ZGO 框架三个核心增强模块的需求：事件系统（Event System）、模块化组织（Modular Organization）和测试工具（Testing Utilities）。这些增强旨在提升框架的可观测性、代码边界清晰度和测试体验。

## Glossary

- **Event_System**: 领域事件的发布-订阅系统，支持同步和异步事件分发
- **Event_Bus**: 事件总线，负责事件的路由和分发
- **Event_Handler**: 事件处理器，响应特定事件的回调函数
- **Event_Subscriber**: 事件订阅者，注册监听特定事件类型的组件
- **Module_Registry**: 模块注册表，管理所有已注册模块的元数据
- **Module_Boundary**: 模块边界，定义模块的公开接口和内部实现
- **Test_Suite**: 测试套件，组织和运行相关测试的容器
- **Test_Fixture**: 测试夹具，提供测试所需的预配置环境
- **Mock_Builder**: Mock 构建器，用于创建测试替身的工具
- **Assertion_Helper**: 断言助手，提供丰富的测试断言方法

## Requirements

### Requirement 1: Event Publishing

**User Story:** As a developer, I want to publish domain events from my services, so that other parts of the system can react to business changes.

#### Acceptance Criteria

1. WHEN a service calls `Publish(ctx, event)`, THE Event_System SHALL deliver the event to all registered handlers
2. WHEN publishing an event, THE Event_System SHALL include event metadata (timestamp, event_id, correlation_id)
3. WHEN the context is cancelled during publishing, THE Event_System SHALL stop delivery and return a context error
4. IF an event handler returns an error during synchronous dispatch, THEN THE Event_System SHALL return the error to the publisher
5. THE Event_System SHALL support both synchronous and asynchronous event dispatch modes

### Requirement 2: Event Subscription

**User Story:** As a developer, I want to subscribe to domain events, so that I can implement reactive business logic.

#### Acceptance Criteria

1. WHEN a handler calls `Subscribe(eventName, handler)`, THE Event_System SHALL register the handler for that event type
2. WHEN multiple handlers subscribe to the same event, THE Event_System SHALL invoke all handlers in registration order
3. WHEN a handler is registered with priority, THE Event_System SHALL invoke handlers in priority order (higher first)
4. THE Event_System SHALL support wildcard subscriptions (e.g., "user.*" matches "user.created", "user.deleted")
5. WHEN `Unsubscribe(eventName, handler)` is called, THE Event_System SHALL remove the handler from the subscription list

### Requirement 3: Event Middleware

**User Story:** As a developer, I want to add cross-cutting concerns to event handling, so that I can implement logging, tracing, and error handling consistently.

#### Acceptance Criteria

1. WHEN middleware is registered, THE Event_System SHALL execute middleware in order before the handler
2. THE Event_System SHALL support middleware for logging event dispatch with configurable log levels
3. THE Event_System SHALL support middleware for distributed tracing integration (OpenTelemetry)
4. THE Event_System SHALL support middleware for error recovery with configurable retry policies
5. WHEN middleware calls `next()`, THE Event_System SHALL continue to the next middleware or handler

### Requirement 4: Event Persistence (Optional)

**User Story:** As a developer, I want to persist events for audit and replay, so that I can implement event sourcing patterns.

#### Acceptance Criteria

1. WHERE event persistence is enabled, THE Event_System SHALL store events to the configured storage backend
2. WHERE event persistence is enabled, THE Event_System SHALL support querying historical events by type, time range, and correlation_id
3. WHERE event replay is requested, THE Event_System SHALL replay events in chronological order

### Requirement 5: Module Registration

**User Story:** As a developer, I want to register modules with the framework, so that they can be discovered and managed centrally.

#### Acceptance Criteria

1. WHEN a module calls `Register(module)`, THE Module_Registry SHALL store the module metadata
2. THE Module_Registry SHALL validate that module names are unique
3. IF a duplicate module name is registered, THEN THE Module_Registry SHALL return an error
4. THE Module_Registry SHALL support querying modules by name, tag, or dependency
5. WHEN the application starts, THE Module_Registry SHALL initialize modules in dependency order

### Requirement 6: Module Lifecycle

**User Story:** As a developer, I want modules to have clear lifecycle hooks, so that I can manage resources properly.

#### Acceptance Criteria

1. WHEN a module implements `OnInit(ctx)`, THE Module_Registry SHALL call it during application startup
2. WHEN a module implements `OnStart(ctx)`, THE Module_Registry SHALL call it after all modules are initialized
3. WHEN a module implements `OnStop(ctx)`, THE Module_Registry SHALL call it during graceful shutdown (reverse order)
4. IF a module's lifecycle hook returns an error, THEN THE Module_Registry SHALL log the error and continue with other modules
5. THE Module_Registry SHALL support timeout configuration for lifecycle hooks

### Requirement 7: Module Dependencies

**User Story:** As a developer, I want to declare module dependencies, so that the framework can ensure proper initialization order.

#### Acceptance Criteria

1. WHEN a module declares dependencies via `DependsOn()`, THE Module_Registry SHALL initialize dependencies first
2. IF a circular dependency is detected, THEN THE Module_Registry SHALL return an error at registration time
3. THE Module_Registry SHALL support optional dependencies that don't block initialization
4. WHEN querying a module's dependencies, THE Module_Registry SHALL return both direct and transitive dependencies

### Requirement 8: Module Boundaries

**User Story:** As a developer, I want clear module boundaries, so that I can maintain separation of concerns.

#### Acceptance Criteria

1. THE Module_Boundary SHALL define a public API interface for inter-module communication
2. THE Module_Boundary SHALL hide internal implementation details from other modules
3. WHEN a module exposes services, THE Module_Boundary SHALL register them with the DI container
4. THE Module_Boundary SHALL support versioned APIs for backward compatibility

### Requirement 9: Test Suite Builder

**User Story:** As a developer, I want a fluent API for building test suites, so that I can write tests more efficiently.

#### Acceptance Criteria

1. THE Test_Suite SHALL support grouping related tests with `Describe()` and `It()` blocks
2. THE Test_Suite SHALL support setup and teardown hooks (`BeforeEach`, `AfterEach`, `BeforeAll`, `AfterAll`)
3. THE Test_Suite SHALL support skipping tests with `Skip()` and focusing tests with `Focus()`
4. THE Test_Suite SHALL support table-driven tests with `TestCases()` builder
5. WHEN a test fails, THE Test_Suite SHALL provide detailed failure messages with context

### Requirement 10: Test Fixtures

**User Story:** As a developer, I want reusable test fixtures, so that I can reduce test setup boilerplate.

#### Acceptance Criteria

1. THE Test_Fixture SHALL support creating pre-configured database connections (in-memory SQLite)
2. THE Test_Fixture SHALL support creating pre-configured HTTP test servers
3. THE Test_Fixture SHALL support automatic cleanup after test completion
4. THE Test_Fixture SHALL support fixture composition for complex test scenarios
5. WHEN a fixture is created, THE Test_Fixture SHALL isolate test data from other tests

### Requirement 11: Mock Builder

**User Story:** As a developer, I want to easily create mocks for testing, so that I can isolate units under test.

#### Acceptance Criteria

1. THE Mock_Builder SHALL support creating mocks from interfaces
2. THE Mock_Builder SHALL support setting return values with `Returns()`
3. THE Mock_Builder SHALL support verifying call counts with `Times(n)`
4. THE Mock_Builder SHALL support argument matching with `WithArgs()`
5. THE Mock_Builder SHALL support capturing arguments for later inspection

### Requirement 12: Assertion Helpers

**User Story:** As a developer, I want rich assertion helpers, so that I can write expressive test assertions.

#### Acceptance Criteria

1. THE Assertion_Helper SHALL support JSON path assertions for API responses
2. THE Assertion_Helper SHALL support database state assertions
3. THE Assertion_Helper SHALL support event emission assertions
4. THE Assertion_Helper SHALL support time-based assertions with tolerance
5. WHEN an assertion fails, THE Assertion_Helper SHALL provide a diff between expected and actual values

### Requirement 13: Event Testing

**User Story:** As a developer, I want to test event-driven code, so that I can verify event publishing and handling.

#### Acceptance Criteria

1. THE Test_Suite SHALL support capturing published events during tests
2. THE Test_Suite SHALL support asserting that specific events were published
3. THE Test_Suite SHALL support asserting event payload contents
4. THE Test_Suite SHALL support asserting event order
5. THE Test_Suite SHALL support mocking event handlers for isolation

### Requirement 14: Integration Test Helpers

**User Story:** As a developer, I want helpers for integration testing, so that I can test module interactions.

#### Acceptance Criteria

1. THE Test_Suite SHALL support spinning up isolated test environments
2. THE Test_Suite SHALL support seeding test data with factories
3. THE Test_Suite SHALL support cleaning up test data between tests
4. THE Test_Suite SHALL support parallel test execution with isolated databases
5. WHEN running integration tests, THE Test_Suite SHALL provide transaction rollback support

