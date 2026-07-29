---
name: frontend-design
description: Design or substantially restyle a Luas page or component. Use for visual-direction work, not small copy, state, data, or bug-only edits.
---

# Luas Frontend Design

## Purpose

Establish a coherent visual direction for a substantial surface while keeping
Luas a product-neutral enterprise scaffold. This skill is not required for a
small component fix or an edit that already has a clear local pattern.

## Start With Context

1. Identify the surface: operational console, public site, authentication,
   devtool, or downstream product page.
2. Identify the repeated workflow and information density users need.
3. Inspect the nearest existing layout, primitives, tokens, and responsive
   behavior before proposing a new direction.
4. Preserve Rhine blue brand ownership and shadcn semantics unless the user
   explicitly requests a new downstream brand.

## Direction Rules

For Luas console and operational tools:

- prioritize scanning, comparison, repeated action, and predictable hierarchy;
- use restrained surfaces, compact headings, stable toolbars, and clear state;
- avoid marketing heroes, decorative card grids, visual noise, and novelty
  motion;
- keep icons, labels, controls, tables, and sidebars aligned at every viewport.

For public or downstream branded surfaces, visual expression may be stronger,
but it must follow the actual audience and content. Do not choose an extreme
aesthetic merely to make the page look distinctive.

## Implementation Workflow

1. State the audience, surface, and one-sentence visual direction.
2. Reuse existing tokens and primitives; add a variant only when it expresses
   a repeatable semantic need.
3. Build the complete workflow, including loading, empty, error, disabled,
   focus, and mobile states.
4. Use real domain content or representative neutral scaffold data.
5. Verify the changed view at desktop and mobile widths; check long localized
   labels and keyboard focus.

## Quality Bar

- Typography matches the information density and never overflows controls.
- Color communicates hierarchy and state with accessible contrast.
- Cards represent real repeated/framed units, not arbitrary page sections.
- Motion explains state change and respects reduced motion.
- Familiar actions use familiar icons and accessible labels.
- Layout dimensions remain stable under loading and translated copy.
- No new visual dependency is added without a concrete need.

## Related Skills

Select another skill only when its distinct concern is active:

- `ui-styling-guide`: token or primitive implementation details.
- `web-design-guidelines`: explicit design/UX review.
- `accessibility-audit`: dedicated WCAG pass.
- `web-perf`: measured visual/runtime performance concern.
