# Luas Shared Control Contracts

Read this reference only when the audit touches shared form controls,
calendar/date picker, password/color controls, or composed buttons and links.

## Form Controls

- `Input` retains native HTML semantics; a `type` value never swaps in another
  component.
- Specialized interactions use explicit `DatePicker`, `ColorPicker`, or
  `PasswordInput` APIs.
- React DayPicker owns calendar-grid semantics and keyboard focus. Custom
  components forward supplied accessibility, event, and focus props.
- Error descriptions preserve caller descriptions by merging
  `aria-describedby` IDs. Errors are announced and associated with the control.
- Icon-only actions receive caller-owned localized labels; a reusable primitive
  does not add an English fallback.
- `src/test/form-control-accessibility.test.tsx` owns native attributes, error
  associations, password visibility labels, and color-control semantics.
- `src/test/calendar-date-picker.test.tsx` owns locale, keyboard, dialog, and
  local form-value behavior.

## Composed Buttons And Links

- Use native `Button` for actions and `Button asChild` with one semantic link
  for navigation.
- The composed link owns the focus ring, complete pointer hit area, accessible
  name, and `data-slot`; icons and loading feedback remain inside it.
- Disabled composed links use `aria-disabled`, leave the tab order, and
  suppress pointer and keyboard activation without receiving invalid native
  `disabled`.
- Primitive-owned disabled state overrides conflicting child ARIA, tab, and
  activation props.
- Loading controls use `aria-busy`; decorative spinners remain hidden from the
  accessibility tree.
- `src/test/button-composition.test.tsx` is the public regression seam for these
  semantics.
