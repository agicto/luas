---
name: accessibility-audit
description: WCAG 2.2 AA accessibility audit pass for React components (focus order, ARIA, contrast, keyboard nav, semantics). Use when adding or reviewing UI.
---

# Accessibility Audit

## Purpose

Make changes inclusive by default. `frontend-design` and `web-design-guidelines` cover aesthetics; this skill covers whether the result is usable by someone with a screen reader, a keyboard, or low vision.

Target: WCAG 2.2 AA. Use the WAI-ARIA Authoring Practices for component patterns.

## When to Use

Before merging any change that:

- adds or restructures a component
- adds an interactive widget (modal, dropdown, tabs, accordion, toast, popover)
- changes navigation, focus flow, or page structure
- changes color, typography, or layout in a way that affects readability

Skip for pure logic / data-layer changes with no rendered output.

## Audit Checklist

### 1. Keyboard

- [ ] Every interactive element is reachable with `Tab`.
- [ ] Focus order matches visual order.
- [ ] `Escape` closes modals, dropdowns, command palettes.
- [ ] `Enter` and `Space` activate buttons; `Enter` submits forms.
- [ ] Arrow keys navigate within composite widgets (menus, tabs, radio groups).
- [ ] There is no keyboard trap — `Tab` always exits.

### 2. Focus Visibility

- [ ] Focus ring is visible on every interactive element. Do not set `outline: none` without a replacement.
- [ ] Focus ring meets 3:1 contrast against the adjacent background.
- [ ] When a modal opens, focus moves into the modal; when it closes, focus returns to the trigger.

### 3. Semantics

- [ ] Use the right element first: `<button>` for actions, `<a>` for navigation, `<input>` for input. Never `<div onClick>` for an action.
- [ ] Headings form a single outline: one `<h1>` per page, no skipped levels.
- [ ] Shared title primitives do not hardcode a heading level; `AlertTitle` is a non-heading container so the page owns its outline.
- [ ] Lists use `<ul>` / `<ol>` / `<li>`.
- [ ] Forms wrap controls in `<label>` or use `aria-labelledby`. Every input has an accessible name.
- [ ] Landmarks exist: `<header>`, `<nav>`, `<main>`, `<footer>`.

### 4. ARIA (only when semantics can't carry it)

- [ ] No redundant ARIA on native elements (`<button role="button">` is wrong).
- [ ] `aria-expanded`, `aria-controls`, `aria-haspopup` on dropdown triggers.
- [ ] `aria-current="page"` on the active nav item.
- [ ] `aria-live="polite"` for non-urgent dynamic updates (toasts, save indicators); `assertive` only for true emergencies.
- [ ] `role="dialog"` + `aria-modal="true"` + `aria-labelledby` for modals.
- [ ] Decorative icons: `aria-hidden="true"`. Functional icons: `aria-label`.

### 5. Contrast

- [ ] Body text: 4.5:1 against background.
- [ ] Large text (≥18pt or 14pt bold): 3:1.
- [ ] UI components and graphical objects: 3:1.
- [ ] Disabled controls are still distinguishable from background (not WCAG-required but good practice).

Use Chrome DevTools color picker or the `axe` extension to check.

Run `pnpm lint:theme-contrast` after changing theme primitives or semantic mappings. It checks supported light/dark text pairs at the WCAG AA 4.5:1 threshold. Use `text-error` for readable error copy and reserve `destructive` for destructive action surfaces or borders.

### 6. Images, Media, Motion

- [ ] Every `<img>` has `alt`. Decorative: `alt=""`. Informative: describe purpose.
- [ ] `AvatarImage` callers choose `alt` explicitly; use an empty value when adjacent visible text already names the person.
- [ ] Videos have captions; audio has transcripts.
- [ ] Respect `prefers-reduced-motion`: gate non-essential animation behind the media query.
- [ ] No autoplaying audio.

### 7. Forms

- [ ] Error messages are announced (`aria-describedby` pointing to the error element).
- [ ] Required fields are programmatically marked (`required` or `aria-required="true"`).
- [ ] Validation errors do not depend on color alone — include text or icon.
- [ ] Error text is associated with the input that failed, not floating elsewhere.

#### Luas Form Primitive Contract

- `Input` must retain native HTML semantics; never replace it with another component based on `type`.
- Use explicit `DatePicker`, `ColorPicker`, or `PasswordInput` APIs for specialized interactions.
- Keep `Calendar` grid semantics and keyboard focus on React DayPicker; custom DayPicker components must forward the supplied accessibility, event, and focus props.
- Preserve caller descriptions when adding error descriptions; merge `aria-describedby` ids.
- Icon-only actions must receive caller-owned accessible labels. Do not add an English fallback inside a reusable primitive.
- `src/test/form-control-accessibility.test.tsx` is the public regression seam for native attributes, error associations, password visibility labels, and color control semantics. `src/test/calendar-date-picker.test.tsx` owns calendar locale, keyboard, dialog, and local form-value contracts.

#### Luas Composed Control Contract

- Use a native `Button` for actions and `Button asChild` with one semantic link for navigation.
- The composed link itself must own the focus ring, complete pointer hit area, accessible name, and
  `data-slot`; icons and loading feedback must remain inside it.
- Disabled composed links use `aria-disabled`, leave the tab order, and suppress pointer and keyboard
  activation without receiving the invalid native `disabled` attribute. Primitive-owned disabled
  state must override conflicting child ARIA, tab, and activation props.
- Loading controls use `aria-busy`; decorative spinners stay hidden from the accessibility tree.
- `src/test/button-composition.test.tsx` is the public regression seam for these semantics.

### 8. Internationalization

- [ ] `lang` attribute set on `<html>` for the current locale (this project's `i18n-handler` skill governs locale).
- [ ] Text containers handle longer translations without overflow or truncation.
- [ ] Right-to-left layouts work if any RTL locale is supported.

## Tools

Run these and triage findings:

- `pnpm dlx @axe-core/cli http://localhost:3000/<page>` — automated audit.
- Chrome DevTools → Lighthouse → Accessibility.
- Tab through the page with no mouse; can you complete every task?
- Open VoiceOver (`Cmd+F5` on macOS) and listen to the first viewport.

## Source Material

Before auditing, read:

1. The shadcn component being used — most ship with correct ARIA already.
2. The `web-design-guidelines` skill for project tokens (colors, focus styles).
3. The `i18n-handler` skill for locale and RTL handling.
4. Existing accessible patterns in `src/features/` to match style.
5. `src/components/ui/form-control.tsx` and `src/test/form-control-accessibility.test.tsx` for form error and accessible-name contracts.

## Output

For each finding:

- Severity: **block** (WCAG AA fail), **fix** (best practice), **note** (style).
- Location: `file:line`.
- Suggested fix in one or two lines.

If clean: "Audited `<component>`. No findings."

## Anti-patterns

- "Looks fine on my screen" — that's not the audit.
- Adding ARIA to fix bad semantics. Fix the semantics.
- Using `tabindex` > 0 to control order. Reorder the DOM instead.
- `aria-label` on a button that already has visible text — usually wrong.
- "We'll add a11y later" — retrofit is 10× the cost of doing it right the first time.

## Pair With

- `frontend-design` and `ui-styling-guide` for the visual side.
- `web-perf` — accessibility and perf often interact (lazy-loading, focus management).
- `verification-before-completion` — accessibility checks belong in Tier 2.
