---
name: testing-standards
description: Write or restructure Web unit, component, and integration tests with Vitest and Testing Library. Use for test implementation or test-boundary decisions.
---

# testing-standards

## Overview

This skill defines testing standards for the project using Vitest and React Testing Library. It ensures consistent, maintainable tests that provide confidence in code changes.

## Guidelines

### 1. Test File Organization

- **Location**: All tests go in `src/test/` directory
- **Naming**: `[feature].test.ts` or `[component].test.tsx`
- **Setup**: Global setup in `src/test/setup.ts`

```
src/test/
├── setup.ts              # Global test configuration
├── utils.test.ts         # Utility function tests
├── hooks/                # Hook tests
│   └── use-auth.test.ts
└── components/           # Component tests
    └── auth-form.test.tsx
```

### 2. Test Commands

| Command | Purpose |
|---------|---------|
| `corepack pnpm vitest run <test-file>` | Run the focused non-watch test |
| `corepack pnpm test -- --run` | Run the complete suite once |
| `corepack pnpm test:coverage` | Run coverage only when requested or required |

### 3. Component Testing Pattern

Use React Testing Library for component tests:

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect } from 'vitest';
import { LoginForm } from '@/features/auth';

describe('LoginForm', () => {
  it('should display error on invalid email', async () => {
    const user = userEvent.setup();
    render(<LoginForm />);
    
    await user.type(screen.getByLabelText(/email/i), 'invalid');
    await user.click(screen.getByRole('button', { name: /submit/i }));
    
    expect(screen.getByText(/invalid email/i)).toBeInTheDocument();
  });
});
```

### 4. Hook Testing Pattern

Test hooks with `renderHook`:

```tsx
import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useLogin } from '@/features/auth/hooks/use-auth';

describe('useLogin', () => {
  it('should call login API with credentials', async () => {
    const { result } = renderHook(() => useLogin());
    
    result.current.mutate({ email: 'test@example.com', password: 'pass' });
    
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
  });
});
```

For shared UI primitives, assert the public DOM and interaction contract rather than private helper
structure. `src/test/button-composition.test.tsx` verifies that a composed link is the semantic host
for styles, focus, icons, loading state, and disabled activation.

### 5. Mocking Guidelines

- **Next.js Router**: Already mocked in `setup.ts`
- **API Calls**: Use `vi.mock()` or MSW for HTTP mocking
- **Zustand Stores**: Create a fresh store per test; auth stores are provider-owned

```tsx
import { vi } from 'vitest';
import { createAuthStore } from '@/features/auth/store/auth-store';

const store = createAuthStore(
  { status: 'client-required' },
  vi.fn().mockResolvedValue({ user: testUser })
);

// Mock feature service
vi.mock('@/features/auth/services/auth-service', () => ({
  authService: {
    login: vi.fn().mockResolvedValue({ id: '1', name: 'Test' }),
  },
}));
```

### 6. Naming Conventions

- **Describe blocks**: Use component/function name
- **It blocks**: Start with "should" + expected behavior
- **Test IDs**: Use `data-testid` sparingly, prefer accessible queries

```tsx
// ✅ Good - accessible query
screen.getByRole('button', { name: /submit/i });

// ⚠️ Fallback - test ID
screen.getByTestId('submit-button');
```

> [!IMPORTANT]
> **Do NOT** place test files alongside source files. All tests must be in `src/test/` directory.

## Related Skills

Select another skill only when its distinct concern is active.

- [`webapp-testing`](../webapp-testing/): Browser-driven complement to unit/component tests.
- [`data-state-management`](../data-state-management/): Mocking state in tests.
- [`systematic-debugging`](../../../../.agents/skills/systematic-debugging/): When a failing test surfaces a bug needing a real root cause.
