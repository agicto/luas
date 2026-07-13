import { ApiErrorCode } from '@/http/codes';
import type { AuthStatus } from '@/features/auth/types';

export type AuthField = 'email' | 'name' | 'password';
export type AuthSessionFailureStatus = Extract<
  AuthStatus,
  'forbidden' | 'unauthenticated' | 'unavailable'
>;

interface AuthFailureLike {
  errorCode?: unknown;
  fieldErrors?: unknown;
  status?: unknown;
}

function asAuthFailure(error: unknown): AuthFailureLike | undefined {
  return typeof error === 'object' && error !== null
    ? (error as AuthFailureLike)
    : undefined;
}

export function getAuthErrorCode(error: unknown): string | undefined {
  const errorCode = asAuthFailure(error)?.errorCode;
  return typeof errorCode === 'string' ? errorCode : undefined;
}

export function getAuthErrorStatus(error: unknown): number | undefined {
  const status = asAuthFailure(error)?.status;
  return typeof status === 'number' ? status : undefined;
}

export function classifyAuthSessionFailure(
  error: unknown
): AuthSessionFailureStatus {
  const errorCode = getAuthErrorCode(error);

  if (
    errorCode === ApiErrorCode.AUTH_FORBIDDEN ||
    errorCode === ApiErrorCode.AUTH_ACCOUNT_DISABLED
  ) {
    return 'forbidden';
  }

  if (
    errorCode === ApiErrorCode.AUTH_UNAUTHORIZED ||
    errorCode === ApiErrorCode.AUTH_INVALID_CREDENTIALS
  ) {
    return 'unauthenticated';
  }

  const status = getAuthErrorStatus(error);

  if (status === 403) {
    return 'forbidden';
  }

  if (status === 401) {
    return 'unauthenticated';
  }

  return 'unavailable';
}

export function hasAuthFieldError(
  error: unknown,
  field: AuthField
): boolean {
  const fieldErrors = asAuthFailure(error)?.fieldErrors;

  if (typeof fieldErrors !== 'object' || fieldErrors === null) {
    return false;
  }

  const messages = (fieldErrors as Record<string, unknown>)[field];
  return Array.isArray(messages) && messages.length > 0;
}

export function isUnauthenticatedAuthError(error: unknown): boolean {
  const errorCode = getAuthErrorCode(error);

  if (
    errorCode === ApiErrorCode.AUTH_FORBIDDEN ||
    errorCode === ApiErrorCode.AUTH_ACCOUNT_DISABLED
  ) {
    return false;
  }

  return (
    errorCode === ApiErrorCode.AUTH_UNAUTHORIZED ||
    getAuthErrorStatus(error) === 401
  );
}
