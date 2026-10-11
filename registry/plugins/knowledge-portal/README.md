# knowledge-portal

Starts an internal or customer-facing knowledge portal: the knowledge base with full-text and Chinese substring search, organization tags, threaded comments with mentions, the warm reading theme, and landing and empty-state blocks. Selects the organization and notification starters so mentions reach people.

## Includes

- `knowledge-base`
- `tags`
- `comments`
- `theme-warm`
- `block-landing-hero`
- `block-empty-state`

## Use

```bash
make init-project NAME="Acme" SLUG=acme MODULE=github.com/acme/app/api TEMPLATE=knowledge-portal
# or, in an existing project:
cd api && go run ./cmd/fastbuild plugin:add knowledge-portal
```

It selects the starters `organization`, `notification`. Complete each included plugin's
INTEGRATION.md steps, then run `plugin:doctor` and `make check`.
