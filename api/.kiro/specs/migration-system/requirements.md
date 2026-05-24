# Requirements Document

## Introduction

This document defines the requirements for refactoring ZGO's database migration system to follow Laravel's elegant design patterns. The goal is to create a comprehensive, maintainable, and extensible migration system that provides:

- Repository pattern for migration record storage
- Core Migrator engine with event-driven architecture
- Fluent Schema Builder with Blueprint DSL
- Full-featured CLI commands with pretend mode
- Proper batch management and rollback capabilities

## Glossary

- **Migrator**: The core engine that coordinates all migration operations (run, rollback, reset, fresh)
- **Migration_Repository**: Interface defining the contract for migration record storage
- **Database_Migration_Repository**: Concrete implementation storing migration records in database
- **Migration**: Interface defining the contract for individual migration files with `Up()` and `Down()` methods
- **Schema_Builder**: Fluent interface for creating and modifying database tables
- **Blueprint**: DSL for defining table columns, indexes, and constraints within a schema operation
- **Batch**: A group of migrations run together in a single operation, identified by a batch number
- **Pretend_Mode**: A mode where migrations show SQL statements without executing them
- **Migration_Event**: Events fired during migration lifecycle (started, ended, skipped)

## Requirements

### Requirement 1: Migration Repository Interface

**User Story:** As a framework developer, I want an abstract interface for migration record storage, so that I can swap implementations without changing the Migrator.

#### Acceptance Criteria

1. THE Migration_Repository interface SHALL define a `GetRan()` method that returns all completed migration names
2. THE Migration_Repository interface SHALL define a `GetMigrations(steps int)` method that returns the last N migrations
3. THE Migration_Repository interface SHALL define a `GetMigrationsByBatch(batch int)` method that returns migrations for a specific batch
4. THE Migration_Repository interface SHALL define a `GetLast()` method that returns migrations from the last batch
5. THE Migration_Repository interface SHALL define a `GetMigrationBatches()` method that returns all migrations with their batch numbers
6. THE Migration_Repository interface SHALL define a `Log(migration string, batch int)` method to record a migration run
7. THE Migration_Repository interface SHALL define a `Delete(migration string)` method to remove a migration record
8. THE Migration_Repository interface SHALL define a `GetNextBatchNumber()` method that returns the next batch number
9. THE Migration_Repository interface SHALL define a `CreateRepository()` method to create the migrations table
10. THE Migration_Repository interface SHALL define a `RepositoryExists()` method to check if the migrations table exists
11. THE Migration_Repository interface SHALL define a `DeleteRepository()` method to drop the migrations table

### Requirement 2: Database Migration Repository Implementation

**User Story:** As a framework developer, I want a database-backed implementation of the migration repository, so that migration records are persisted reliably.

#### Acceptance Criteria

1. THE Database_Migration_Repository SHALL implement the Migration_Repository interface
2. THE Database_Migration_Repository SHALL store migration records in a configurable table (default: "migrations")
3. THE Database_Migration_Repository SHALL store migration name and batch number for each record
4. WHEN `GetRan()` is called, THE Database_Migration_Repository SHALL return migrations ordered by batch ascending, then name ascending
5. WHEN `GetLast()` is called, THE Database_Migration_Repository SHALL return migrations from the highest batch number
6. WHEN `GetNextBatchNumber()` is called, THE Database_Migration_Repository SHALL return the maximum batch number plus one
7. WHEN `CreateRepository()` is called, THE Database_Migration_Repository SHALL create a table with id, migration, and batch columns
8. THE Database_Migration_Repository SHALL support MySQL, PostgreSQL, and SQLite dialects

### Requirement 3: Migration Interface

**User Story:** As a developer, I want a clear contract for migration files, so that I can create consistent and predictable migrations.

#### Acceptance Criteria

1. THE Migration interface SHALL define an `Up(db *gorm.DB) error` method for applying the migration
2. THE Migration interface SHALL define a `Down(db *gorm.DB) error` method for reverting the migration
3. THE Migration interface SHALL define a `GetConnection() string` method to specify the database connection
4. THE Migration interface SHALL define a `WithinTransaction() bool` method to indicate if the migration should run in a transaction
5. WHEN a migration file is created, THE system SHALL generate a file with timestamp prefix (YYYY_MM_DD_HHMMSS_name.go)

### Requirement 4: Migrator Core Engine

**User Story:** As a framework developer, I want a central migration engine, so that all migration operations are coordinated consistently.

#### Acceptance Criteria

1. THE Migrator SHALL accept a Migration_Repository, database connection, and optional event dispatcher
2. WHEN `Run()` is called, THE Migrator SHALL execute all pending migrations in order
3. WHEN `Run()` is called with `pretend: true`, THE Migrator SHALL display SQL statements without executing them
4. WHEN `Run()` is called with `step: true`, THE Migrator SHALL increment batch number for each migration
5. WHEN `Rollback()` is called, THE Migrator SHALL revert the last batch of migrations
6. WHEN `Rollback()` is called with `step: N`, THE Migrator SHALL revert the last N migrations
7. WHEN `Rollback()` is called with `batch: N`, THE Migrator SHALL revert all migrations in batch N
8. WHEN `Reset()` is called, THE Migrator SHALL revert all migrations in reverse order
9. THE Migrator SHALL fire migration events (MigrationStarted, MigrationEnded, MigrationsStarted, MigrationsEnded)
10. THE Migrator SHALL support running migrations within transactions when the migration specifies it
11. IF a migration fails, THEN THE Migrator SHALL return an error with the migration name and failure reason

### Requirement 5: Schema Builder

**User Story:** As a developer, I want a fluent interface for schema operations, so that I can create and modify tables elegantly.

#### Acceptance Criteria

1. THE Schema_Builder SHALL provide a `Create(table string, callback func(*Blueprint))` method for creating tables
2. THE Schema_Builder SHALL provide a `Table(table string, callback func(*Blueprint))` method for modifying tables
3. THE Schema_Builder SHALL provide a `Drop(table string)` method for dropping tables
4. THE Schema_Builder SHALL provide a `DropIfExists(table string)` method for conditionally dropping tables
5. THE Schema_Builder SHALL provide a `Rename(from, to string)` method for renaming tables
6. THE Schema_Builder SHALL provide a `HasTable(table string) bool` method for checking table existence
7. THE Schema_Builder SHALL provide a `HasColumn(table, column string) bool` method for checking column existence
8. THE Schema_Builder SHALL generate dialect-specific SQL for MySQL, PostgreSQL, and SQLite

### Requirement 6: Blueprint DSL

**User Story:** As a developer, I want a fluent DSL for defining table structure, so that I can write readable and maintainable migrations.

#### Acceptance Criteria

1. THE Blueprint SHALL provide column methods: `ID()`, `String()`, `Text()`, `Integer()`, `BigInteger()`, `Boolean()`, `Timestamp()`, `JSON()`
2. THE Blueprint SHALL provide `Timestamps()` method that creates `created_at` and `updated_at` columns
3. THE Blueprint SHALL provide `SoftDeletes()` method that creates a `deleted_at` column
4. THE Blueprint SHALL provide `Primary()`, `Unique()`, `Index()` methods for creating indexes
5. THE Blueprint SHALL provide `Foreign()` method that returns a ForeignKeyDefinition for fluent foreign key creation
6. THE Blueprint SHALL provide `DropColumn()`, `RenameColumn()` methods for column modifications
7. THE Blueprint SHALL provide `DropPrimary()`, `DropUnique()`, `DropIndex()`, `DropForeign()` methods for dropping indexes
8. WHEN a column method is called, THE Blueprint SHALL return a ColumnDefinition for method chaining
9. THE ColumnDefinition SHALL support `Nullable()`, `Default()`, `Unsigned()`, `AutoIncrement()`, `Comment()` modifiers

### Requirement 7: Migration Events

**User Story:** As a framework developer, I want migration events, so that I can hook into the migration lifecycle for logging and monitoring.

#### Acceptance Criteria

1. THE system SHALL fire a `MigrationsStarted` event before running any migrations with direction (up/down)
2. THE system SHALL fire a `MigrationsEnded` event after all migrations complete with direction (up/down)
3. THE system SHALL fire a `MigrationStarted` event before each individual migration with migration name and method
4. THE system SHALL fire a `MigrationEnded` event after each individual migration with migration name and method
5. THE system SHALL fire a `MigrationSkipped` event when a migration is skipped
6. THE system SHALL fire a `NoPendingMigrations` event when there are no migrations to run
7. WHEN events are fired, THE system SHALL use the existing ZGO event bus infrastructure

### Requirement 8: CLI Commands

**User Story:** As a developer, I want comprehensive CLI commands, so that I can manage migrations from the command line.

#### Acceptance Criteria

1. THE `db:migrate` command SHALL run all pending migrations
2. THE `db:migrate` command SHALL support `--pretend` flag to show SQL without executing
3. THE `db:migrate` command SHALL support `--step` flag to run migrations one batch at a time
4. THE `db:migrate` command SHALL support `--force` flag to run in production
5. THE `db:rollback` command SHALL rollback the last batch of migrations
6. THE `db:rollback` command SHALL support `--step=N` flag to rollback N migrations
7. THE `db:rollback` command SHALL support `--batch=N` flag to rollback a specific batch
8. THE `db:rollback` command SHALL support `--pretend` flag to show SQL without executing
9. THE `db:reset` command SHALL rollback all migrations
10. THE `db:fresh` command SHALL drop all tables and re-run all migrations
11. THE `db:fresh` command SHALL support `--seed` flag to run seeders after migrations
12. THE `db:status` command SHALL display a table showing migration name, batch, and status (Ran/Pending)
13. THE `make:migration` command SHALL create a new migration file with timestamp prefix
14. THE `make:migration` command SHALL support `--create=table` flag to generate create table stub
15. THE `make:migration` command SHALL support `--table=table` flag to generate modify table stub

### Requirement 9: Migration File Creator

**User Story:** As a developer, I want automatic migration file generation, so that I can quickly scaffold new migrations.

#### Acceptance Criteria

1. WHEN `make:migration create_users_table --create=users` is called, THE system SHALL generate a migration with `Schema.Create()` stub
2. WHEN `make:migration add_email_to_users --table=users` is called, THE system SHALL generate a migration with `Schema.Table()` stub
3. THE generated migration file SHALL follow the naming convention: `YYYY_MM_DD_HHMMSS_migration_name.go`
4. THE generated migration file SHALL be placed in the `database/migrations/` directory
5. THE generated migration file SHALL include proper package declaration and imports
6. THE generated migration file SHALL implement the Migration interface with `Up()` and `Down()` methods
7. THE system SHALL automatically register the migration in the migrations registry

### Requirement 10: Pretend Mode

**User Story:** As a developer, I want to preview migration SQL, so that I can verify changes before applying them.

#### Acceptance Criteria

1. WHEN `--pretend` flag is used, THE system SHALL capture all SQL statements that would be executed
2. WHEN `--pretend` flag is used, THE system SHALL NOT execute any SQL statements against the database
3. WHEN `--pretend` flag is used, THE system SHALL display each migration name followed by its SQL statements
4. THE pretend mode SHALL work for both `db:migrate` and `db:rollback` commands
5. THE pretend mode SHALL show SQL statements in the order they would be executed

