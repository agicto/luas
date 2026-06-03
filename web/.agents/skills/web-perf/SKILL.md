---
name: web-perf
description: Core Web Vitals audit (LCP, INP, CLS) and Next.js perf review. Use when investigating slowness, before shipping perf-sensitive pages, or when LCP/CLS regress.
---

# Web Performance Audit

## Purpose

Measure before optimizing. This skill structures perf work around the same metrics Google ranks on and users feel: LCP, INP, CLS. Without measurement, optimization is theater.

## When to Use

- A user reports the app feels slow.
- A page in the perf-sensitive tier (landing, dashboard, checkout) is being shipped.
- Lighthouse score regresses in CI.
- LCP / INP / CLS exceed targets in real-user monitoring.

Skip for internal admin pages or B2B back-office views where the perf budget is relaxed.

## Targets (WCV "Good")

| Metric | Target |
|---|---|
| LCP (Largest Contentful Paint) | < 2.5s |
| INP (Interaction to Next Paint) | < 200ms |
| CLS (Cumulative Layout Shift) | < 0.1 |
| FCP (First Contentful Paint) | < 1.8s |
| TTFB (Time to First Byte) | < 800ms |

Measure at the 75th percentile of real users, not on your laptop.

## Audit Workflow

### Phase 1: Measure

- [ ] Run Lighthouse against the page in incognito with throttling on (Slow 4G, 4× CPU slowdown).
- [ ] Capture the LCP element, INP interaction, and any layout shift sources.
- [ ] Compare against the targets above.
- [ ] If you have field data (Vercel Speed Insights, RUM), use that — it's what users experience.

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

- [ ] Re-run Lighthouse; record the delta.
- [ ] Confirm the metric you targeted actually moved (an LCP fix should reduce LCP — not just overall score).
- [ ] Verify no regression in other metrics. INP fixes sometimes hurt LCP.
- [ ] If shipping to prod, set up alerting on the metric (Vercel Speed Insights or equivalent).

## Anti-patterns

- "I added `loading=\"lazy\"` to everything" — lazy-loading above the fold *hurts* LCP.
- "I added `useMemo` everywhere" — `useMemo` has its own cost; profile before adding.
- "I shipped `'use client'` for everything because it's simpler" — every client boundary is hydration cost.
- Optimizing on a M2 with fiber to gigabit — that's not your user.
- Comparing builds on localhost — real perf wins show up only with realistic network/CPU.
- Pursuing a Lighthouse score without checking field data — synthetic ≠ real.

## Tools

- Chrome DevTools → Performance panel (record an interaction).
- Lighthouse (local + Vercel).
- `@next/bundle-analyzer` for bundle size.
- React DevTools Profiler for component-level cost.
- Vercel Speed Insights for field data.

## Source Material

Before auditing, read:

1. `vercel-react-best-practices` skill for code patterns (memoization, RSC boundaries).
2. The page's data-fetching strategy (`async` Server Component vs client-side `useSWR`).
3. `next.config.js` and any custom webpack config.
4. `web-design-guidelines` for any global font / image conventions.

## Output

Report:

1. Current numbers (LCP / INP / CLS) at p75.
2. Biggest single contributor identified.
3. Fix applied and the delta.
4. What you did *not* fix and why (out of scope, low impact, tracked separately).

## Pair With

- `vercel-react-best-practices` for code-pattern fixes.
- `accessibility-audit` — perf and a11y often interact (focus management, lazy-loading).
- `verification-before-completion` so the perf claim is grounded in numbers, not vibes.
