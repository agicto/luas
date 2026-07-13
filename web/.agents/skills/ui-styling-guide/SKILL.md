---
name: ui-styling-guide
description: Guidelines for UI development, including shadcn/ui primitives, feature components, and the OKLCH theme system.
---

# ui-styling-guide

## Overview

This skill defines the standards for UI development within the project. It ensures visual consistency, accessibility, and maintainability by using a structured design system based on OKLCH and shadcn/ui.

## Guidelines

### 1. Theme System (OKLCH & CSS Variables)

The project uses a layered Design Token system. Always prefer semantic classes over raw colors.

- **Primitives**: Base colors in `src/themes/primitives.css` (Do not use directly).
- **Semantic Tokens**: Functional naming in `light.css` / `dark.css`.
  - Backgrounds: `bg-canvas`, `bg-surface`, `bg-subtle`
  - Foregrounds: `text-main`, `text-subtle`, `text-muted`
  - Borders: `border-main`, `border-subtle`, `border-strong`
  - Brand: `brand-main`, `brand-subtle`, `brand-strong`

```tsx
<div className="bg-bg-surface text-text-main border-border-subtle shadow-md p-4 rounded-lg">
  <h1 className="text-brand">Heading</h1>
  <p className="text-text-subtle">Subtle text description.</p>
</div>
```

### 2. Component Organization

- **UI Primitives**: `src/components/ui/` contains project-owned, shadcn-derived primitives. Modify them only at the shared ownership seam, preserve public APIs and accessibility behavior, and add contract tests for behavioral changes. Review any shadcn CLI overwrite as a migration, not a mechanical refresh.
- **Feature Components**: prefer `src/features/[feature]/components/` for feature-owned UI.
- **Shared Feature UI Blocks**: use `src/components/features/` only for reusable cross-feature UI.
- **Common Components**: `src/components/common/` (Generic, non-business specific).

### 3. Component Standards

For detailed component contracts (Named Exports, RSC First, Props Typing, CAS annotations), refer to:

> **[AGENTS.md - Atomic Component Contract](file:///AGENTS.md#atomic-component-contract)**

### 4. File Naming

- Components: `kebab-case.tsx` (e.g., `login-form.tsx`)
- Hooks: `use-kebab-case.ts` (e.g., `use-mobile.ts`)

> [!IMPORTANT]
> **Localized Core Copy**: Formal site, auth, and console surfaces must use `getT` or `useT` for user-facing copy. Follow the i18n handler's core-copy boundary for exact brands, technical identifiers, disposable devtools/examples, and the root fallback. Run `pnpm lint:i18n-copy` before completion.

### Form Controls

- Keep `Input` native. Specialized controls such as `DatePicker`, `ColorPicker`, and `PasswordInput` must be imported explicitly.
- Use the shared form-control error seam for stable ids, `aria-invalid`, merged `aria-describedby`, and polite announcements.
- Require caller-owned labels for icon-only actions so formal surfaces can localize them.
- Run `pnpm exec vitest run src/test/form-control-accessibility.test.tsx` after changing shared form controls.

## Related Skills

- [`frontend-design`](../frontend-design/): Creative direction that styling implements.
- [`web-design-guidelines`](../web-design-guidelines/): Design tokens this guide consumes.
- [`accessibility-audit`](../accessibility-audit/): Focus styles, contrast, and ARIA in styling decisions.
- [`i18n-handler`](../i18n-handler/): Layout must accommodate variable translation lengths.
