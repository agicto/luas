---
name: web-design-guidelines
description: Review Luas UI code against interface guidelines. Use for explicit design, UX, or visual-quality review; use accessibility-audit for a dedicated WCAG pass.
---

# Web Interface Guidelines

Review files for compliance with Web Interface Guidelines.

## How It Works

1. Fetch the latest guidelines from the source URL below
2. Read the specified files (or prompt user for files/pattern)
3. Check against all rules in the fetched guidelines
4. Output findings in the terse `file:line` format

## Guidelines Source

Fetch fresh guidelines before each review:

```
https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md
```

Use WebFetch to retrieve the latest rules. The fetched content contains all the rules and output format instructions.

## Usage

When a user provides a file or pattern argument:
1. Fetch guidelines from the source URL above
2. Read the specified files
3. Apply all rules from the fetched guidelines
4. Output findings using the format specified in the guidelines

If no files specified, ask the user which files to review.

## Related Skills

Select another skill only when its distinct concern is active.

- [`frontend-design`](../frontend-design/): Creative direction these tokens serve.
- [`ui-styling-guide`](../ui-styling-guide/): Tailwind / shadcn mechanics that consume these tokens.
- [`accessibility-audit`](../accessibility-audit/): Verify color / focus tokens meet WCAG AA contrast.
- [`web-perf`](../web-perf/): CSS perf when applying tokens at scale.
