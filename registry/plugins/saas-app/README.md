# saas-app

Starts a subscription-style SaaS product: organizations with roles and permissions, usage metering, settings, and notifications, styled with the enterprise theme, with blocks for the landing page, pricing, sign-in, onboarding, dashboard, data tables, settings, and empty states. Billing stays a product decision and is not included.

## Includes

- `theme-enterprise`
- `block-landing-hero`
- `block-pricing-table`
- `block-auth-card`
- `block-onboarding-steps`
- `block-dashboard-stats`
- `block-data-table`
- `block-settings-page`
- `block-empty-state`

## Use

```bash
make init-project NAME="Acme" SLUG=acme MODULE=github.com/acme/app/api TEMPLATE=saas-app
# or, in an existing project:
cd api && go run ./cmd/fastbuild plugin:add saas-app
```

It selects the starters `organization`, `permission`, `notification`, `setting`, `usage`. Complete each included plugin's
INTEGRATION.md steps, then run `plugin:doctor` and `make check`.
