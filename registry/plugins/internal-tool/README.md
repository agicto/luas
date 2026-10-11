# internal-tool

Starts a back-office or operations tool: organizations with roles and permissions, the operator Admin Console, tags and threaded comments for records, and dashboard, data table, settings, and empty-state blocks in the enterprise theme.

## Includes

- `tags`
- `comments`
- `theme-enterprise`
- `block-dashboard-stats`
- `block-data-table`
- `block-settings-page`
- `block-empty-state`

## Use

```bash
make init-project NAME="Acme" SLUG=acme MODULE=github.com/acme/app/api TEMPLATE=internal-tool
# or, in an existing project:
cd api && go run ./cmd/fastbuild plugin:add internal-tool
```

It selects the starters `organization`, `permission`, `operator`. Complete each included plugin's
INTEGRATION.md steps, then run `plugin:doctor` and `make check`.
