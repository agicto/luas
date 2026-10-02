---
name: utility-tooling
description: Add a reusable Web utility or hook after repository search. Use for genuinely shared pure helpers, not feature-owned behavior or one-off wrappers.
---

# utility-tooling

## Overview

Keep the shared helper surface small. `src/utils` currently exports only `cn`, and `src/hooks`
holds the two cross-feature hooks. A helper earns a place there only when at least two features
need the same pure behavior.

## Guidelines

### 1. Search first

Before writing a utility or hook:

1. Check `src/utils/index.ts` and `src/hooks/`.
2. Check the owning feature under `src/features/<feature>/`; behavior used by one feature stays
   there.
3. Check native Web APIs: `Intl`, `URL`, `URLSearchParams`, `crypto`, `structuredClone`.
4. Check dependencies already in `package.json`. Do not add a date, collection, or validation
   library for one call; schemas use Zod.

### 2. Placement

- Shared pure function: `src/utils/<name>.ts`, exported from `src/utils/index.ts`, with a test.
- Shared hook: `src/hooks/use-<purpose>.ts`, following the Rules of Hooks.
- Anything that reads environment, cookies, or the network is not a utility; it belongs to
  `src/config`, `src/http`, or a feature service.

### 3. Verification

```bash
corepack pnpm vitest run <test-file>
corepack pnpm type-check
corepack pnpm lint
```

## Related Skills

- [`environment-config`](../environment-config/): environment access and validation.
