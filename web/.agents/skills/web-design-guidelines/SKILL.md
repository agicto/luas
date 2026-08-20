---
name: web-design-guidelines
description: Review Luas UI code against interface guidelines. Use for explicit design, UX, or visual-quality review; use accessibility-audit for a dedicated WCAG pass.
---

# Web Interface Review

Review a completed Luas UI for design and UX quality. This skill reports
findings; it does not establish a new visual direction or run a WCAG audit.

## Review Authority

Use the checked-in sources first:

1. `web/AGENTS.md` for Luas UI, responsive, i18n, and component rules.
2. The nearest existing route, feature, primitive, and theme tokens.
3. The requested files and their loading, empty, error, disabled, and mobile
   states.

Do not fetch third-party guidelines during an ordinary review. If the user
explicitly requests comparison with the latest Vercel Web Interface Guidelines,
fetch their current primary source and label those findings as external rather
than Luas requirements.

## Review Axes

- **Hierarchy**: the primary task, information density, headings, grouping, and
  action priority are scannable.
- **Interaction**: controls use familiar semantics, clear labels, predictable
  feedback, and complete hit areas.
- **States**: loading, empty, error, disabled, success, and destructive states
  preserve layout and explain the next action.
- **Responsive behavior**: content reflows without clipping, accidental
  horizontal scrolling, or hidden primary actions.
- **System consistency**: existing tokens, primitives, variants, spacing, and
  icon conventions are reused instead of creating a parallel design system.
- **Content**: copy is concise, localized, domain-accurate, and does not expose
  backend or provider details.

Baseline keyboard, focus, naming, and contrast rules still apply through
`web/AGENTS.md`. Use `accessibility-audit` only when the request is a dedicated
WCAG/a11y review.

## Output

Return only actionable findings ordered by severity:

```text
severity file:line - problem; smallest useful correction
```

Use **block** for a broken primary workflow, **fix** for meaningful UX/design
debt, and **note** for a lower-risk polish opportunity. If no findings remain,
state the reviewed scope and say so explicitly.

## Related Skills

Navigation only; do not load automatically:

- `frontend-design` for a new visual direction.
- `ui-styling-guide` for implementing tokens and primitives.
- `accessibility-audit` for an explicit WCAG pass.
