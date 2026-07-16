# Contributing To Luas Web

The project-wide contribution policy lives in [../CONTRIBUTING.md](../CONTRIBUTING.md). Read it together with [AGENTS.md](AGENTS.md) and [docs/ADDING_FEATURE.md](docs/ADDING_FEATURE.md) before changing the Web app.

For Web changes, also run the relevant local checks:

```bash
corepack pnpm type-check
corepack pnpm lint
corepack pnpm test -- --run
corepack pnpm build
```

Keep browser features under `src/features/`, use server-only adapters for credentials and upstream access, preserve mock and production contract parity, and update i18n messages and tests with user-visible behavior.
