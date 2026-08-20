---
name: accessibility-audit
description: Audit a completed Luas UI against WCAG 2.2 AA. Use when the user requests accessibility, a11y, or WCAG review, not for routine UI implementation.
---

# Accessibility Audit

Audit a completed surface against WCAG 2.2 AA through code inspection,
automated evidence, and manual interaction. Baseline accessibility remains
mandatory through `web/AGENTS.md`; this explicit workflow is not an automatic
follow-up to ordinary UI work.

## Scope And Authority

Identify the pages, components, user workflows, supported locales, and changed
interaction states before auditing. Read:

1. `web/AGENTS.md` and the target code.
2. The underlying shadcn/Radix primitive when custom composition changes its
   semantic or focus contract.
3. Existing public tests for the same interaction.
4. WAI-ARIA Authoring Practices only for the specific composite widget being
   reviewed.

When shared form controls, calendar/date picker, password/color controls, or
composed buttons/links are in scope, also read
[references/luas-controls.md](references/luas-controls.md). Do not load other
design skills merely because they are adjacent to accessibility.

## Audit Procedure

### Semantics and accessible names

- Prefer native elements: buttons perform actions, links navigate, labels name
  controls, and lists/headings/landmarks express document structure.
- Keep one page-owned `h1` and a logical heading outline. Shared title
  primitives remain heading-neutral.
- Every control has one accurate accessible name. Decorative icons are hidden;
  informative images have purposeful alt text and redundant images use
  `alt=""`.
- Add ARIA only where native semantics cannot express the state. Validate
  widget relationships such as `aria-expanded`, `aria-controls`,
  `aria-current`, dialog labelling, and live-region priority.

### Keyboard and focus

- Complete every primary workflow without a pointer.
- Tab order follows DOM and visual order; no positive `tabindex` or keyboard
  trap exists.
- Composite widgets implement their expected arrow-key behavior. Enter, Space,
  and Escape behave according to the control pattern.
- Focus is visibly at least 3:1 against adjacent colors. Dialogs move focus in
  and restore it to the trigger; route/state changes place focus deliberately.

### Forms and dynamic state

- Required state, instructions, validation errors, and descriptions are
  programmatically associated with their controls and never rely on color
  alone.
- Status changes use an appropriate live region without duplicating visible
  labels or announcing routine updates aggressively.
- Loading controls expose busy state; disabled behavior remains perceivable and
  cannot activate through pointer or keyboard paths.

### Perception, motion, and layout

- Normal text meets 4.5:1 contrast; large text, component boundaries, focus,
  and meaningful graphics meet 3:1 where WCAG requires it.
- Zoom/reflow, long translations, validation messages, and mobile widths do not
  clip content or hide the primary action.
- Non-essential motion respects `prefers-reduced-motion`; media has the
  required captions or transcript and no autoplaying audio.
- The document `lang` matches the current locale. Test RTL only when the
  supported locale set includes it.

## Evidence

Use the narrowest evidence available. Run these commands from `web/`:

```bash
# Theme contrast changes
corepack pnpm lint:theme-contrast

# Shared Luas control contracts
corepack pnpm vitest run \
  src/test/form-control-accessibility.test.tsx \
  src/test/calendar-date-picker.test.tsx \
  src/test/button-composition.test.tsx

# A running page, when runtime evidence is part of the request
corepack pnpm dlx @axe-core/cli http://localhost:3000/<page>
```

Automated tools do not prove keyboard order, focus restoration, accessible
names in context, screen-reader clarity, or completion of the real workflow.
Tab through the surface and use VoiceOver or another available screen reader
for the changed path.

## Output

Return actionable findings with severity, WCAG rationale, `file:line`, and the
smallest correction:

- **block**: a WCAG AA failure or inaccessible primary workflow.
- **fix**: a material accessibility defect or fragile custom interaction.
- **note**: a non-blocking best-practice improvement.

If clean, state the audited code/runtime scope and the manual and automated
evidence used. Do not claim WCAG conformance from automated output alone.

## Completion Criteria

- Primary workflows complete with keyboard and screen-reader interaction.
- Semantics, names, focus, state announcements, contrast, reflow, and motion
  have direct evidence.
- Relevant Luas control-contract tests pass.
- Every finding points to an observable failure and a scoped correction.

## Related Skills

Navigation only; do not load automatically:

- `frontend-design` for a separate visual redesign request.
- `web-perf` for a separate measured performance regression.
