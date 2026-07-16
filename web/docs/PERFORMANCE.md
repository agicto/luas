# Web Performance Boundary

This document defines how Luas measures and guards Web performance. It separates a repeatable
build-time regression signal from local synthetic measurements and real-user outcomes.

## Canonical Terms

**First-load JavaScript** is `firstLoadUncompressedJsBytes` from Next.js 16's generated
`.next/diagnostics/route-bundle-stats.json`. It is the deduplicated JavaScript set Next reports for
loading one route. Luas enforces the uncompressed value because it is deterministic across build
runs; gzip bytes are useful diagnostics but are not the gate.

**Route performance budget** is the reviewed upper bound in
[`../performance-budgets.json`](../performance-budgets.json). It is an executable regression guard,
not a target to consume and not a Core Web Vital or service-level objective.

**Synthetic performance evidence** is a controlled local or CI measurement such as Lighthouse.
Use at least three comparable production runs and report the median plus the test conditions.

**Field Web Vitals** are real production-user measurements evaluated at the 75th percentile. The
current good thresholds are LCP at or below 2.5 seconds, INP at or below 200 milliseconds, and CLS
at or below 0.1. A local Lighthouse result cannot establish these field outcomes.

## Build Gate

`pnpm build` creates the production build and then runs
[`../scripts/check-route-bundle-budget.mjs`](../scripts/check-route-bundle-budget.mjs). The checker:

- reads the official Next.js route diagnostics rather than scraping console output;
- rejects missing, duplicate, malformed, or path-escaping route evidence;
- enforces named route budgets plus a global maximum;
- reports gzip level-9 totals for diagnosis without using them as the pass/fail signal.

The baseline captured on 2026-07-16 with Next.js 16.2.9 is:

| Route | Budget, raw bytes | Measured raw bytes | Measured gzip bytes |
|---|---:|---:|---:|
| `/` | 600,000 | 579,149 | 168,775 |
| `/login` | 900,000 | 866,083 | 262,605 |
| `/register` | 900,000 | 866,083 | 262,605 |
| `/console` | 925,000 | 884,030 | 268,755 |
| `/console/organizations/[organizationId]` | 1,070,000 | 1,023,736 | 305,892 |

Run the supported tools from `web/`:

```bash
pnpm build
pnpm bundle:check
pnpm bundle:analyze
```

`bundle:check` reuses an existing production build. `bundle:analyze` writes the official Turbopack
analysis output without starting an analyzer server.

The production Docker build copies `pnpm-workspace.yaml` with `package.json` and the lockfile so
frozen installs see the same reviewed security overrides. It also preserves the optional `public/`
asset seam when a downstream app has no public files yet. Its builder runs `pnpm build`, which means
the image cannot bypass route budgets:

```bash
docker build --tag luas-web:local .
```

## Change Procedure

1. Run a clean production build before changing a provider, root layout, shared control, analytics
   integration, large dependency, or route-level lazy boundary.
2. Record the affected route's raw bytes, gzip bytes, and chunk count.
3. Make one attributable change, rebuild, and compare the same route set.
4. Run the relevant interaction and responsive checks; a smaller bundle does not excuse broken UI.
5. Update a budget only for an intentional route capability. Include before/after evidence and the
   owning feature in the same review. Do not raise a budget solely to make the gate pass.
6. Keep field claims separate. Add production RUM through an explicit downstream adapter when a
   product chooses its observability vendor and privacy policy.

Prefer Server Components and leaf client boundaries, but measure the generated graph. A dynamic
import is not automatically cheaper: wrapping the optional analytics server component in a dynamic
boundary added about 1,995 raw bytes to every measured route and was reverted. Google Analytics
uses `lazyOnload` only to defer optional third-party execution when configured; it is not credited
with a default bundle reduction.

For a local lab comparison, start the production server and run Lighthouse three times with the
same browser, viewport, CPU, and network settings:

```bash
pnpm exec next start -H 127.0.0.1 -p 3110
CHROME_PATH=/path/to/chromium pnpm dlx lighthouse@12.8.2 http://127.0.0.1:3110/ \
  --only-categories=performance \
  --output=json \
  --output-path=/tmp/luas-home.json \
  --chrome-flags="--headless --no-sandbox"
```

The 2026-07-16 three-run mobile Lighthouse median changed as follows after replacing the public
theme menu with one hydration-stable native three-state selector:

| Signal | Before | After | Interpretation |
|---|---:|---:|---|
| Performance score | 94 | 96 | Synthetic summary only |
| FCP | 906 ms | 905 ms | One-millisecond difference; no meaningful change |
| LCP | 3,072 ms | 2,855 ms | 217 ms lower; still above the field-good threshold |
| CLS | 0 | 0 | Stable in the lab |
| TBT | 8 ms | 3.5 ms | 4.5 ms lower in the lab |
| Transfer bytes | 289.6 kB | 248.3 kB | 14.2% lower |
| Main-thread work | 340.6 ms | 307.7 ms | 9.7% lower |

The comparable raw runs behind those medians were:

| Phase | Run | Score | FCP ms | LCP ms | CLS | TBT ms | Transfer bytes | Main thread ms |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Before | 1 | 93 | 909.37 | 3,223.37 | 0 | 14 | 289,467 | 329.57 |
| Before | 2 | 94 | 906.04 | 3,072.04 | 0 | 8 | 289,582 | 361.78 |
| Before | 3 | 94 | 905.73 | 3,070.73 | 0 | 7.5 | 289,586 | 340.57 |
| After | 1 | 98 | 910.23 | 2,486.23 | 0 | 3.5 | 248,314 | 418.94 |
| After | 2 | 95 | 905.20 | 2,920.20 | 0 | 15 | 248,638 | 294.19 |
| After | 3 | 96 | 905.15 | 2,855.15 | 0 | 0 | 247,936 | 307.67 |

The public `/` route's first-load JavaScript moved from 696,759 raw / 207,561 gzip bytes and 16
chunks to 579,149 raw / 168,775 gzip bytes and 12 chunks. These measurements are local synthetic
and build evidence; Luas does not yet ship a production RUM adapter or claim field p75 results.

## References

- [Next.js package and route bundle analysis](https://nextjs.org/docs/pages/guides/package-bundling)
- [Next.js production checklist](https://nextjs.org/docs/app/guides/production-checklist)
- [Next.js Script strategies](https://nextjs.org/docs/pages/api-reference/components/script)
- [Core Web Vitals](https://web.dev/articles/vitals)
- [Core Web Vitals thresholds](https://web.dev/articles/defining-core-web-vitals-thresholds)
