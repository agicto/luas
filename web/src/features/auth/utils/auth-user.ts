import type { AuthUser } from '@/features/auth/types';

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}

export function isAuthUser(value: unknown): value is AuthUser {
  if (typeof value !== 'object' || value === null) {
    return false;
  }

  const user = value as Record<string, unknown>;

  return (
    isNonEmptyString(user.id) &&
    isNonEmptyString(user.email) &&
    isNonEmptyString(user.name) &&
    (user.role === 'admin' || user.role === 'member')
  );
}
