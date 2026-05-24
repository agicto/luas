# Implementation Plan: Migration System

## Overview

This implementation plan refactors ZGO's database migration system to follow Laravel's elegant design patterns. The implementation is organized into phases: core infrastructure, schema builder, migrator engine, CLI commands, and migration file creator.

## Tasks

- [x] 1. Set up migration package structure and core interfaces
  - Create `internal/infra/migration/` directory structure
  - Define `Repository` interface with all methods
  - Define `Migration` interface with `Up()`, `Down()`, `GetConnection()`, `WithinTransaction()`
  - Create `BaseMigration` struct with default implementations
  - Define `MigrationRecord` and `RollbackOptions` types
  - _Requirements: 1.1-1.11, 3.1-3.4_

- [x] 2. Implement Database Migration Repository
  - [x] 2.1 Implement `databaseRepository` struct with configurable table name
    - Implement `NewDatabaseRepository(db, tableName)` constructor
    - Implement `GetRan()` with correct ordering (batch ASC, migration ASC)
    - Implement `GetMigrations(steps)` for last N migrations
    - Implement `GetMigrationsByBatch(batch)` for specific batch
    - Implement `GetLast()` using `getLastBatchNumber()`
    - _Requirements: 2.1-2.5_

  - [x] 2.2 Implement repository management methods
    - Implement `Log(migration, batch)` to record migration
    - Implement `Delete(migration)` to remove record
    - Implement `GetNextBatchNumber()` returning max+1
    - Implement `GetMigrationBatches()` returning map[string]int
    - _Requirements: 2.6-2.8_

  - [x] 2.3 Implement repository lifecycle methods
    - Implement `CreateRepository()` to create migrations table
    - Implement `RepositoryExists()` to check table existence
    - Implement `DeleteRepository()` to drop migrations table
    - Support MySQL, PostgreSQL, SQLite dialects
    - _Requirements: 2.7, 2.8, 1.9-1.11_

  - [x] 2.4 Write property tests for Repository
    - **Property 1: Repository GetRan Ordering**
    - **Property 2: Repository GetLast Batch Filtering**
    - **Property 3: Repository Next Batch Number**
    - **Property 4: Migration Log Round-Trip**
    - **Validates: Requirements 2.3-2.6**

- [x] 3. Implement Schema Builder foundation
  - [x] 3.1 Create Grammar interface and implementations
    - Define `Grammar` interface with `Compile(*Blueprint) []string`
    - Implement `MySQLGrammar` for MySQL-specific SQL
    - Implement `PostgresGrammar` for PostgreSQL-specific SQL
    - Implement `SQLiteGrammar` for SQLite-specific SQL
    - Create `NewGrammar(dialect string)` factory function
    - _Requirements: 5.8_

  - [x] 3.2 Implement Blueprint DSL
    - Create `Blueprint` struct with table, columns, commands
    - Implement `Create()` to mark as creating new table
    - Implement column methods: `ID()`, `String()`, `Text()`, `Integer()`, `BigInteger()`, `Boolean()`, `Timestamp()`, `JSON()`
    - Implement `Timestamps()` for created_at/updated_at
    - Implement `SoftDeletes()` for deleted_at
    - _Requirements: 6.1-6.3_

  - [x] 3.3 Implement ColumnDefinition with method chaining
    - Create `ColumnDefinition` struct
    - Implement `Nullable()`, `Default()`, `Unsigned()`, `AutoIncrement()`, `Primary()`, `Comment()` modifiers
    - Ensure all modifiers return `*ColumnDefinition` for chaining
    - _Requirements: 6.8, 6.9_

  - [x] 3.4 Implement index and foreign key methods
    - Implement `Primary()`, `Unique()`, `Index()` for creating indexes
    - Implement `Foreign()` returning `*ForeignKeyDefinition`
    - Create `ForeignKeyDefinition` with `References()`, `On()`, `OnDelete()`, `OnUpdate()`, `Cascade()`, `Constrained()`
    - Implement `DropColumn()`, `RenameColumn()`, `DropPrimary()`, `DropUnique()`, `DropIndex()`, `DropForeign()`
    - _Requirements: 6.4-6.7_

  - [x] 3.5 Implement SchemaBuilder
    - Create `Builder` struct with db and grammar
    - Implement `Create(table, callback)` for creating tables
    - Implement `Table(table, callback)` for modifying tables
    - Implement `Drop()`, `DropIfExists()`, `Rename()`
    - Implement `HasTable()`, `HasColumn()`
    - _Requirements: 5.1-5.7_

  - [x] 3.6 Write property tests for Schema Builder
    - **Property 13: Schema HasTable Reflects Existence**
    - **Validates: Requirements 5.6**

- [x] 4. Checkpoint - Ensure schema layer tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement Migration Events
  - [x] 5.1 Create migration event types
    - Create `MigrationsStarted` event with Direction field
    - Create `MigrationsEnded` event with Direction field
    - Create `MigrationStarted` event with Migration and Method fields
    - Create `MigrationEnded` event with Migration and Method fields
    - Create `MigrationSkipped` event with Migration field
    - Create `NoPendingMigrations` event with Direction field
    - Implement `Name()` method for each event type
    - _Requirements: 7.1-7.6_

  - [x] 5.2 Write unit tests for events
    - Test event name methods return correct strings
    - Test event fields are properly set
    - _Requirements: 7.1-7.6_

- [x] 6. Implement Migrator Core Engine
  - [x] 6.1 Create Migrator struct and constructor
    - Create `Migrator` struct with repository, db, events, migrations map
    - Implement `NewMigrator(repo, db, eventBus)` constructor
    - Implement `Register(name, migration)` to add migrations
    - Implement `getPendingMigrations(ran)` helper
    - _Requirements: 4.1_

  - [x] 6.2 Implement Run operation
    - Implement `Run(opts MigratorOptions)` method
    - Get pending migrations by comparing registered vs ran
    - Execute migrations in order, logging each to repository
    - Support `step` option to increment batch per migration
    - Fire `MigrationsStarted`, `MigrationStarted`, `MigrationEnded`, `MigrationsEnded` events
    - Fire `NoPendingMigrations` when nothing to run
    - _Requirements: 4.2, 4.4, 4.9_

  - [x] 6.3 Implement Rollback operation
    - Implement `Rollback(opts RollbackOptions)` method
    - Support rollback by last batch (default)
    - Support rollback by steps (`--step=N`)
    - Support rollback by specific batch (`--batch=N`)
    - Delete migration records after successful rollback
    - Fire appropriate events
    - _Requirements: 4.5, 4.6, 4.7_

  - [x] 6.4 Implement Reset operation
    - Implement `Reset(pretend bool)` method
    - Get all ran migrations and reverse order
    - Execute Down() for each in reverse order
    - Delete all migration records
    - _Requirements: 4.8_

  - [x] 6.5 Implement Pretend mode
    - Implement `pretendToRun(migration, method)` method
    - Capture SQL statements without executing
    - Use GORM's DryRun mode or custom SQL capture
    - Return captured SQL for display
    - _Requirements: 4.3, 10.1-10.5_

  - [x] 6.6 Implement transaction support
    - Check `migration.WithinTransaction()` before running
    - Wrap migration execution in `db.Transaction()` if true
    - Handle transaction rollback on error
    - _Requirements: 4.10_

  - [x] 6.7 Implement error handling
    - Wrap migration errors with migration name context
    - Return `fmt.Errorf("migration %s failed: %w", name, err)`
    - _Requirements: 4.11_

  - [x] 6.8 Write property tests for Migrator
    - **Property 5: Migrator Run Executes Pending**
    - **Property 6: Migrator Rollback By Batch**
    - **Property 7: Migrator Rollback By Steps**
    - **Property 8: Migrator Reset Reverses All**
    - **Property 9: Migrator Step Mode Batch Increment**
    - **Property 10: Pretend Mode No Side Effects**
    - **Property 11: Event Ordering**
    - **Property 12: Migration Error Contains Name**
    - **Validates: Requirements 4.2-4.11**

- [x] 7. Checkpoint - Ensure migrator tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 8. Implement CLI Commands
  - [x] 8.1 Refactor db:migrate command
    - Add `--pretend` flag support
    - Add `--step` flag support
    - Add `--force` flag for production
    - Use new Migrator instead of gormigrate
    - Display migration names as they run
    - _Requirements: 8.1-8.4_

  - [x] 8.2 Refactor db:rollback command
    - Add `--step=N` flag support
    - Add `--batch=N` flag support
    - Add `--pretend` flag support
    - Use new Migrator.Rollback()
    - Display rolled back migration names
    - _Requirements: 8.5-8.8_

  - [x] 8.3 Implement db:reset command
    - Create new `DBResetCommand` struct
    - Call Migrator.Reset()
    - Add `--force` flag for production
    - Display reset progress
    - _Requirements: 8.9_

  - [x] 8.4 Refactor db:fresh command
    - Use new Migrator for fresh operation
    - Drop all tables then run migrations
    - Support `--seed` flag
    - Support `--force` flag
    - _Requirements: 8.10, 8.11_

  - [x] 8.5 Refactor db:status command
    - Use new Repository.GetMigrationBatches()
    - Display table with Migration, Batch, Status columns
    - Show "Ran" or "Pending" status
    - _Requirements: 8.12_

  - [x] 8.6 Register new commands
    - Register db:reset command in command registry
    - Update existing command registrations
    - _Requirements: 8.1-8.12_

- [x] 9. Implement Migration File Creator
  - [x] 9.1 Create migration stub templates
    - Create `stubs/migration.stub` for blank migration
    - Create `stubs/migration.create.stub` for create table
    - Create `stubs/migration.update.stub` for modify table
    - Include proper package, imports, struct, Up/Down methods
    - _Requirements: 9.5, 9.6_

  - [x] 9.2 Implement make:migration command
    - Create `MakeMigrationCommand` struct
    - Parse migration name from arguments
    - Generate timestamp prefix (YYYY_MM_DD_HHMMSS)
    - Support `--create=table` flag for create stub
    - Support `--table=table` flag for update stub
    - _Requirements: 8.13-8.15, 9.1-9.3_

  - [x] 9.3 Implement file generation
    - Generate filename with timestamp prefix
    - Replace placeholders in stub template
    - Write file to `database/migrations/` directory
    - Auto-register migration in registry
    - _Requirements: 9.4, 9.7_

  - [x] 9.4 Write property tests for file creator
    - **Property 14: Migration Filename Format**
    - **Validates: Requirements 3.5, 8.13, 9.3**

- [x] 10. Checkpoint - Ensure all CLI commands work
  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. Integration and migration of existing migrations
  - [x] 11.1 Update existing migration files
    - Convert existing gormigrate migrations to new Migration interface
    - Update `database/migrations/migrations.go` registry
    - Ensure backward compatibility with existing migrations table
    - _Requirements: 3.1-3.5_

  - [x] 11.2 Update bootstrap integration
    - Update `internal/bootstrap/migrate.go` to use new Migrator
    - Remove dependency on gormigrate package
    - Integrate with existing event bus
    - _Requirements: 4.1, 7.7_

  - [x] 11.3 Wire dependency injection
    - Create provider for migration components
    - Add to wiring configuration
    - _Requirements: 4.1_

- [x] 12. Final checkpoint - Full integration test
  - Ensure all tests pass, ask the user if questions arise.
  - Test complete migration workflow: create → migrate → rollback → fresh
  - Verify event firing and logging

## Notes

- All tasks are required for comprehensive testing
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (14 properties)
- Unit tests validate specific examples and edge cases
- Use `gopter` library for property-based testing in Go

