---
name: ui-styling-guide
description: Apply Luas UI primitives, variants, Tailwind, and OKLCH tokens. Use when styling within the existing design system, not for broad visual direction.
---

# Luas UI Styling

Implement an already-decided visual direction through the existing Web design
system. Use `frontend-design` instead when audience, hierarchy, or visual
direction is still being designed.

## Authority

Read `web/AGENTS.md` for component ownership, accessibility, i18n, and naming.
Inspect the nearest component and theme files before introducing a token,
variant, primitive, or shared component.

## Tokens

- Use semantic classes backed by `src/themes/light.css` and `dark.css`; do not
  consume primitives from `src/themes/primitives.css` directly in components.
- Prefer canvas/surface/subtle backgrounds, main/subtle/muted foregrounds,
  semantic borders, brand tokens, and status tokens over raw palette values.
- Use `text-error` for readable error copy. Reserve `destructive` for
  destructive action surfaces and borders.
- Change light and dark mappings together. After token changes run
  `corepack pnpm lint:theme-contrast`; guarded normal-text pairs remain at
  least 4.5:1.

## Component Ownership

- `src/components/ui/` owns project-wide shadcn-derived primitives. Treat a
  shadcn CLI overwrite as a reviewed migration and preserve public semantics.
- Feature UI belongs in `src/features/<feature>/components/`.
- `src/components/features/` is for genuinely cross-feature composed UI;
  `src/components/common/` remains generic and product-neutral.
- Add a variant only for a repeatable semantic state. One-off layout belongs at
  the caller.

## Interaction Contracts

- Keep `Input` native. Import `DatePicker`, `ColorPicker`, and `PasswordInput`
  explicitly for specialized behavior.
- Style `Calendar` through React DayPicker slots without replacing grid or
  focus semantics.
- Preserve stable form-control IDs, `aria-invalid`, merged
  `aria-describedby`, and polite error announcements.
- Caller-owned labels are required for icon-only actions; reusable primitives
  do not invent English fallback copy.
- Shared title primitives remain heading-neutral so pages own the document
  outline.
- `AvatarImage` callers choose `alt`; use `alt=""` when adjacent text already
  identifies the person.

## Verification

Run the narrowest check for the changed seam. Run these commands from `web/`:

```bash
# Theme mapping
corepack pnpm lint:theme-contrast

# Shared form and composed-control semantics
corepack pnpm vitest run \
  src/test/form-control-accessibility.test.tsx \
  src/test/calendar-date-picker.test.tsx \
  src/test/button-composition.test.tsx
```

Also run type-check or the owning feature test when a public component API or
caller changes. Use `accessibility-audit` only for an explicitly requested
WCAG pass, not as an automatic styling follow-up.

## Completion Criteria

- Styling uses existing semantic tokens and the correct ownership seam.
- Shared primitive behavior and accessibility contracts remain intact.
- Loading, disabled, focus, localization, and responsive states are stable.
- Focused token/component checks pass.

## Related Skills

Navigation only; do not load automatically:

- `frontend-design` for unresolved visual direction.
- `i18n-handler` for translation-boundary changes.
- `accessibility-audit` for an explicit WCAG review.
