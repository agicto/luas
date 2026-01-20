---
name: ui-styling-guide
description: Guidelines for UI development, including shadcn/ui primitives, feature components, named exports, CAS/CASp annotations, and the OKLCH theme system.
---

# ui-styling-guide

## Overview

This skill defines the standards for UI development within the project. It ensures visual consistency, accessibility, and maintainability by using a structured design system based on OKLCH, shadcn/ui, and strict component contracts.

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
- **UI Primitives**: `src/components/ui/` (shadcn/ui managed, **DO NOT MODIFY**).
- **Feature Components**: `src/components/features/[module]/` (Organized by business module).
- **Common Components**: `src/components/common/` (Generic, non-business specific).

### 3. Atomic Component Contract
All components MUST follow these rules:
- **Named Exports**: Never use `export default`.
- **Strict Typing**: Define props interface named `[ComponentName]Props`.
- **RSC First**: Use Server Components by default. Extract interactivity into small, leaf-level Client Components (`"use client"`).
- **Icon Consistency**: Use `lucide-react` with `size-4` (16px) or `size-5` (20px).

### 4. Component Annotation Standard (CAS)
Include a JSDoc header for all components:

```typescript
/**
 * @component [Formal Name]
 * @category [UI | Feature | Common]
 * @status [Stable | Beta | Experimental]
 * @description [Concise purpose]
 * @usage [When/How to use]
 */
```

## File Naming
- Components: `kebab-case.tsx` (e.g., `login-form.tsx`)
- Hooks: `use-kebab-case.ts` (e.g., `use-mobile.ts`)

> [!IMPORTANT]
> **Zero Hardcoded Strings**: All user-facing text must use the `useT` hook for i18n.
