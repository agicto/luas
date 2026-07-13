import type { AuthResponse, LogoutResponse } from '@/features/auth/types';
import { isAuthUser } from '@/features/auth/utils/auth-user';

export function isAuthResponse(value: unknown): value is AuthResponse {
  return (
    typeof value === 'object' &&
    value !== null &&
    isAuthUser((value as Record<string, unknown>).user)
  );
}

export function isLogoutResponse(value: unknown): value is LogoutResponse {
  return (
    typeof value === 'object' &&
    value !== null &&
    (value as Record<string, unknown>).success === true
  );
}
