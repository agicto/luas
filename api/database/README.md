# Database Migrations

Luas uses timestamped, versioned PostgreSQL migrations. Each migration is a Go type with `Up` and
`Down` methods, registered by `init()`, and owned by one starter manifest. Seeders are described in
[SEEDERS.md](SEEDERS.md); database runtime policy lives in [../docs/DATABASE.md](../docs/DATABASE.md).

## Rules

1. **A released migration is history.** Express its DDL as SQL (run through `execStatements`) or with
   the schema builder, never with `AutoMigrate` on a module's live persistence struct. A later model
   edit must not change what an applied migration creates.
2. **One rollout concern per migration.** Keep expansion, backfill, and destructive contraction in
   separate migrations. Review rollout risk with the `sql-migration-review` skill.
3. **Every migration implements `Down`.** A full reset must return the database to an empty schema.
4. **Never edit an applied migration** except for a schema-neutral fix proven by the golden test.
   Change behavior with a new migration.

## Layout

```text
database/migrations/
├── migrations.go                 # init()-populated registry
├── sql.go                        # execStatements helper for frozen DDL
├── 2025_06_18_000000_create_users_table.go
├── ...
├── schema_golden_postgres_test.go
├── frozen_history_test.go
└── testdata/schema.golden.sql    # schema a fresh database must receive
```

Filenames follow `YYYY_MM_DD_HHMMSS_description.go`; the timestamp orders execution.

## Writing A Migration

```bash
./luas make:migration create_posts_table --create=posts
```

The generator writes a registered type. Fill in `Up` and `Down` with frozen DDL:

```go
func init() {
	register("2026_10_01_000000_create_posts_table", &createPostsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createPostsTable struct {
	migration.BaseMigration
}

// Up creates the posts table.
func (m *createPostsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE posts (
		    id bigserial NOT NULL,
		    user_id bigint NOT NULL,
		    title varchar(200) NOT NULL,
		    created_at timestamptz NOT NULL,
		    CONSTRAINT posts_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_posts_user_created ON posts (user_id, created_at DESC)`,
		`ALTER TABLE posts ADD CONSTRAINT fk_posts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`,
	)
}

// Down drops the posts table.
func (m *createPostsTable) Down(db *gorm.DB) error {
	return execStatements(db, `DROP TABLE IF EXISTS posts CASCADE`)
}
```

`CREATE INDEX CONCURRENTLY` cannot run in a transaction; such a migration sets
`UseTransaction: false` (see `2026_07_25_000000_add_audit_retention_index.go`).

## Ownership

A migration runs only when its starter is active. The owning module lists it in its manifest:

```go
assembly.WithStarterMigrationNames("2026_10_01_000000_create_posts_table")
```

HTTP startup, `db:migrate`, and seeders resolve the same `OPTIONAL_STARTERS` selection, so every
replica and pre-deploy job must use an identical selection.

## Commands

| Command | Alias | Purpose |
|---|---|---|
| `db:migrate [--pretend] [--step] [--force]` | `migrate` | Run pending migrations |
| `db:rollback [--step=N] [--batch=N]` | `migrate:rollback` | Roll back the last batch or N steps |
| `db:status` | `migrate:status` | Show ran and pending migrations |
| `db:reset` | `migrate:reset` | Roll back every migration |
| `db:fresh [--seed]` | `migrate:fresh` | Drop all tables and migrate again |

Destructive commands refuse to run in production without `--force`. Executed migrations are
recorded in the `migrations` table with their batch number.

## Verification

The migration package ships three guards:

- `TestMigrationsProduceGoldenSchema` compares a freshly migrated database with
  `testdata/schema.golden.sql`.
- `TestMigrationsRollBackToEmptySchema` proves every `Down` reverses its `Up`.
- `TestMigrationsDoNotDependOnLiveModelPackages` forbids importing module or capability packages.

```bash
LUAS_TEST_POSTGRES_DSN=postgres://... go test ./database/migrations
```

When a change adds a migration, regenerate the golden file and review its diff as the schema change:

```bash
LUAS_UPDATE_GOLDEN_SCHEMA=1 LUAS_TEST_POSTGRES_DSN=postgres://... \
  go test ./database/migrations -run TestMigrationsProduceGoldenSchema
```
