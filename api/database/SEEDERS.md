# Database Seeders

Seeders load development and demo data. They are not schema history: unlike migrations, they may
use current module persistence structs and must be safe to run repeatedly. Schema changes and data
that every environment requires belong in [migrations](README.md).

## Interface

```go
type Seeder interface {
	Name() string
	Run(db *gorm.DB) error
}
```

Each seeder registers itself in `init()` with `register(&YourSeeder{})`. The default `user` starter
ships `UserSeeder` (`users`), which creates `admin@example.com` and `user@example.com` with the
password `secret` for local development only.

## Writing A Seeder

```bash
./luas make:seeder Product
```

Implement `Run` idempotently, keyed by a unique column, and return every error:

```go
func (s *ProductSeeder) Run(db *gorm.DB) error {
	for _, p := range []catalog.ProductPO{{SKU: "demo-001", Name: "Demo product"}} {
		if err := db.FirstOrCreate(&p, catalog.ProductPO{SKU: p.SKU}).Error; err != nil {
			return err
		}
	}
	return nil
}
```

## Ownership

A seeder runs only when its starter is active. The owning module lists it by name in its manifest:

```go
assembly.WithStarterSeederNames("products")
```

`db:seed` and `db:fresh --seed` resolve seeders from the same `OPTIONAL_STARTERS` selection as HTTP
and migrations, in starter dependency order. Do not rely on file or registration order.

## Commands

| Command | Alias | Purpose |
|---|---|---|
| `db:seed` | `seed` | Run the seeders of the active starters |
| `db:fresh --seed` | `migrate:fresh --seed` | Rebuild the schema, then seed |

Never seed production credentials or secrets. Production bootstrap data belongs in a reviewed
migration or an operator command.
