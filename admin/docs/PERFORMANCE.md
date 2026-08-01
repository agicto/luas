# Admin Console Performance

## Build Budgets

`pnpm build` measures compressed production assets:

| Metric                        |  Budget |
| ----------------------------- | ------: |
| Total JavaScript gzip         | 300 KiB |
| Largest JavaScript chunk gzip | 180 KiB |
| Total CSS gzip                |  50 KiB |

The build also fails when source maps or a server bundle appear. Budget changes
must include before/after production measurements and a clear user-facing
reason.

## Implementation Rules

- Keep TanStack Router automatic code splitting enabled.
- Route entries import feature pages; avoid importing all optional features
  through the root layout.
- Use TanStack Query for remote cache behavior instead of duplicate global
  stores.
- Keep provider composition small and stable.
- Prefer native platform APIs and existing dependencies over another runtime
  package.
- Import icons by name and avoid broad barrel imports for large libraries.
- Treat `index.html` as revalidated and content-hashed assets as immutable.

## Evidence

Build budgets are synthetic engineering evidence, not field Web Vitals.
Production applications should collect LCP, INP, and CLS at the 75th
percentile under an explicit device/reporting policy before making user
performance claims.

Use a bundle analyzer only when a shared dependency or route regression needs
diagnosis. Do not add analyzer runtime to the default production graph.
