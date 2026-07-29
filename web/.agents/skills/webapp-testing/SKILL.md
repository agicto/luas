---
name: webapp-testing
description: Verify a running Luas UI through a real browser. Use for requested workflows, screenshots, responsive checks, or runtime DOM/console inspection, not routine component tests.
---

# Web Application Testing

## Purpose

Verify behavior that unit/component tests and static builds cannot prove:
navigation, responsive layout, focus flow, browser APIs, runtime console errors,
and rendered visual state.

## Tool Routing

1. Reuse the browser capability already available in the active agent
   environment. Do not create a second Playwright process when a connected
   browser can inspect the target.
2. Reuse an already running local server when it belongs to the current
   workspace. Confirm the URL before treating another project as Luas.
3. Start only the deployable unit needed for the workflow. Keep the process
   alive while verifying and stop it unless the user needs the URL afterward.
4. Use `scripts/with_server.py` only for a self-contained headless CLI or CI
   flow when no managed browser/server lifecycle is available. Run its
   `--help` output instead of reading the helper source.

Do not generate a new Python automation file for a one-off check when direct
browser inspection can perform it.

## Workflow

1. **Orient**
   - Confirm the server, URL, viewport, locale, theme, and required auth state.
   - Inspect a DOM snapshot or screenshot before choosing selectors.
2. **Exercise**
   - Prefer accessible roles, names, labels, and stable test attributes.
   - Perform the shortest workflow that reaches the changed behavior.
3. **Observe**
   - Check the authoritative visible state, URL, DOM property, or console entry.
   - Use screenshots when visual layout is the behavior under test.
4. **Vary only relevant conditions**
   - Test desktop/mobile, light/dark, or multiple locales only when the change
     can differ across that condition.
5. **Finish**
   - Report the URL, viewport, workflow, and observed result.
   - Clean up temporary tabs, processes, screenshots, and generated files.

## Evidence Rules

- A screenshot proves appearance, not keyboard behavior or API correctness.
- A DOM assertion proves structure, not visual spacing or clipping.
- Browser navigation is UX evidence; API authorization remains server-owned.
- Do not use fixed sleeps when a visible state or load condition is available.
- Do not inspect broad page content repeatedly; narrow to the changed surface.
- Never place credentials or private payloads in automation source or output.

## Headless Helper

For a repeatable CLI-only flow:

```bash
python web/.agents/skills/webapp-testing/scripts/with_server.py --help
```

The examples directory contains optional Python references. Load one only when
building a persistent headless flow, not for ordinary interactive verification.

## Related Skills

Select another skill only when its distinct concern is active:

- `testing-standards`: unit/component test ownership is unclear.
- `accessibility-audit`: the user requested a dedicated WCAG review.
- Root `systematic-debugging`: runtime behavior fails for an unknown reason.
