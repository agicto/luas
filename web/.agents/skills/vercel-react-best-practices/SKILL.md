---
name: vercel-react-best-practices
description: Apply a specific React/Next.js performance pattern to measured or high-risk code. Use after identifying a waterfall, bundle, serialization, or render problem, not for routine JSX.
---

# React And Next.js Performance Patterns

## Purpose

Route an identified performance problem to the smallest relevant rule. This
skill is a pattern library, not a mandatory checklist for every component.

## Trigger Gate

Use it when evidence or architecture indicates one of these risks:

- sequential async work on a critical request path;
- a large dependency or shared client bundle;
- excessive server-to-client serialization;
- repeated expensive renders or global event listeners;
- an explicit React/Next.js performance review.

Do not load it for routine JSX, styling, copy, forms, or a local component with
no performance signal. Use `web-perf` first when the problem itself has not
been measured or isolated.

## Focused Rule Routing

| Problem | Read |
|---|---|
| Sequential async work | `rules/async-*.md` |
| Bundle growth or optional heavy UI | `rules/bundle-*.md` |
| Server caching or serialized props | `rules/server-*.md` |
| Client fetching or global listeners | `rules/client-*.md` |
| Re-render churn | `rules/rerender-*.md` |
| Long-list or hydration rendering | `rules/rendering-*.md` |
| Proven hot JavaScript loop | `rules/js-*.md` |
| Stable callback edge case | `rules/advanced-*.md` |

Read one or two matching rule files. Do not preload the complete rule catalog.
Use [`references/full-guide.md`](references/full-guide.md) only for an explicit
multi-category audit.

## Workflow

1. Name the observed metric, trace, bundle, or code-path risk.
2. Read the nearest implementation and the smallest matching rule file.
3. Confirm the rule fits Luas ownership and React 19/Next.js 16 behavior.
4. Implement the narrow change without speculative memoization or caching.
5. Re-run the same measurement or deterministic budget that identified the
   problem.

## Guardrails

- Prefer parallel ownership and smaller client boundaries before memoization.
- Do not add cross-request caches without lifecycle, privacy, and bounds.
- Do not replace TanStack Query with another fetching library to follow a
  generic example.
- Preserve Server Component, accessibility, error, and loading semantics.
- A code-pattern change is not a performance improvement until evidence moves.

## Related Skills

- `web-perf`: measurement and route/Web Vital evidence.
- `data-state-management`: Query/Zustand ownership changes.
- `frontend-design`: visual direction constrained by a proven budget.
