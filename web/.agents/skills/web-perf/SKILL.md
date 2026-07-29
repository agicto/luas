---
name: web-perf
description: Measure a reported or suspected Next.js performance regression. Use for bundle budgets, lab evidence, or LCP/INP/CLS analysis, not routine shared-component edits.
---

# Web Performance Audit

## Purpose

Measure before optimizing. This skill keeps three evidence classes explicit: deterministic route
bundle diagnostics, repeatable synthetic/lab measurements, and production field Web Vitals. Each
answers a different question and must be reported under its own name.

## When to Use

- A user reports the app feels slow.
- A page in the perf-sensitive tier (landing, dashboard, checkout) is being shipped.
- A root provider, shared client control, large dependency, analytics script, or lazy boundary changes.
- The route bundle budget or a repeated Lighthouse median regresses.
- LCP / INP / CLS exceed targets in real-user monitoring.

Internal and back-office routes may own higher reviewed budgets, but they still require measurement.

## Field Targets (Core Web Vitals "Good")

| Metric                          | Target   |
| ------------------------------- | -------- |
| LCP (Largest Contentful Paint)  | <= 2.5s  |
| INP (Interaction to Next Paint) | <= 200ms |
| CLS (Cumulative Layout Shift)   | <= 0.1   |

Evaluate these at the 75th percentile of real production users under a declared reporting policy.
FCP, TTFB, TBT, transfer bytes, and main-thread time remain useful diagnostic signals, not Core Web
Vitals. Local Lighthouse cannot establish field p75.

## Audit Workflow

### Phase 1: Measure

- [ ] Read [`../../../docs/PERFORMANCE.md`](../../../docs/PERFORMANCE.md) and identify the route owner.
- [ ] Run a clean `pnpm build`; record route raw bytes, reported gzip bytes, and chunk count.
- [ ] Use `pnpm bundle:analyze` when the changed dependency is not obvious from route evidence.
- [ ] Run Lighthouse against the production server at least three times with identical settings and
      report the median. Capture the LCP element and layout-shift or long-task sources.
- [ ] Record field p75 LCP/INP/CLS when a downstream app has production RUM. Otherwise state that
      field evidence is unavailable.

Do not optimize before you have a number to beat.

### Phase 2: Identify

Common culprits, in order of impact:

**LCP**

- Render-blocking JS / CSS in `<head>`.
- LCP element behind client-side fetch (`useEffect` data load).
- Unoptimized images (no `next/image`, wrong `sizes`, no `priority` on above-the-fold).
- Font CSS without `font-display: swap`.

**INP**

- Long tasks on the main thread (> 50ms).
- Event handlers doing heavy work synchronously.
- React re-renders triggered by every keystroke (missing memoization, context overuse).
- Large hydration cost — too much client-component code shipped.

**CLS**

- Images without `width` / `height` or `aspect-ratio`.
- Ads / embeds / iframes without reserved space.
- Fonts swapping in and causing reflow (use `next/font` with `size-adjust`).
- Dynamic content injected above existing content.

### Phase 3: Fix

Pick the single biggest contributor and fix it. Re-measure. Repeat.

Standard fixes for Next.js 16 / React 19:

**LCP fixes**

- `<Image priority />` on the hero image; pass explicit `sizes`.
- Move data fetching to Server Components.
- `next/font` for self-hosted fonts.
- Preload critical resources via the Next.js metadata API.

**INP fixes**

- Split client bundles: convert components to RSC where possible.
- Defer non-critical scripts (`<Script strategy="lazyOnload" />`).
- `useDeferredValue` / `useTransition` for non-urgent updates.
- Memoize expensive children: `React.memo`, `useMemo` for derived data.
- Move heavy work off-main-thread (Web Worker for parsing, image work).

**CLS fixes**

- Always set `width` and `height` on `<img>`; `next/image` enforces this.
- Reserve space for dynamically loaded content.
- Use `min-height` on containers whose children load asynchronously.

### Phase 4: Verify

- [ ] Re-run `pnpm build`; the executable route budget must pass.
- [ ] Re-run the same Lighthouse protocol; record all three runs and the median delta.
- [ ] Confirm the metric you targeted actually moved (an LCP fix should reduce LCP — not just overall score).
- [ ] Verify no regression in other metrics. INP fixes sometimes hurt LCP.
- [ ] Verify the changed interaction at desktop and mobile widths, including horizontal overflow.
- [ ] If the downstream product owns RUM, compare field p75 after rollout and alert through its chosen
      vendor. Luas itself stays vendor-neutral.

## Anti-patterns

- "I added `loading=\"lazy\"` to everything" — lazy-loading above the fold _hurts_ LCP.
- "I added `useMemo` everywhere" — `useMemo` has its own cost; profile before adding.
- "I shipped `'use client'` for everything because it's simpler" — every client boundary is hydration cost.
- "I wrapped it in `dynamic()` so it must be smaller" — measure; the wrapper can add runtime/chunk overhead.
- "I raised the budget because the feature is intentional" — document ownership and before/after evidence first.
- Optimizing on a M2 with fiber to gigabit — that's not your user.
- Comparing one Lighthouse run — run variance can be larger than the change.
- Pursuing a Lighthouse score without checking field data — synthetic ≠ real.

## Tools

- Chrome DevTools → Performance panel (record an interaction).
- `pnpm build` / `pnpm bundle:check` for official Next.js route diagnostics and budgets.
- `pnpm bundle:analyze` for Next.js 16 Turbopack analysis output.
- Lighthouse for controlled synthetic comparisons.
- React DevTools Profiler for component-level cost.
- A downstream-owned RUM provider for field data.

## Source Material

Before auditing, read:

1. [`../../../docs/PERFORMANCE.md`](../../../docs/PERFORMANCE.md) for Luas terminology, budgets, and commands.
2. `vercel-react-best-practices` skill for code patterns (memoization, RSC boundaries).
3. The page's data-fetching strategy (`async` Server Component vs client-side fetching).
4. `next.config.ts` and the current official analyzer output.
5. `web-design-guidelines` for global font, image, and responsive conventions.

## Output

Report:

1. Route, raw/gzip/chunk evidence, configured budget, and headroom.
2. Lab protocol, all runs, median LCP/CLS/TBT/transfer/main-thread evidence, and biggest contributor.
3. Field p75 LCP/INP/CLS with source and reporting policy, or an explicit "not available" statement.
4. Fix applied, attributable delta, responsive/interaction verification, and reverted experiments.
5. What you did _not_ fix and why (out of scope, low impact, or tracked separately).

## Related Skills

Select another skill only when its distinct concern is active.

- `vercel-react-best-practices` for code-pattern fixes.
- `accessibility-audit` when the user also requests a dedicated WCAG review.
