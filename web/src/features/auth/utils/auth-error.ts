import type { AllTranslationKeys } from '@/i18n';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import {
  getAuthErrorCode,
  getAuthErrorStatus,
} from '@/features/auth/utils/auth-failure';

export {
  hasAuthFieldError,
  isUnauthenticatedAuthError,
} from '@/features/auth/utils/auth-failure';

export type AuthAction = 'login' | 'register' | 'logout';

type AuthErrorKey = Extract<
  AllTranslationKeys,
  | 'auth.accountDisabled'
  | 'auth.emailAlreadyExists'
  | 'auth.invalidCredentials'
  | 'auth.loginFailed'
  | 'auth.logoutFailed'
  | 'auth.registerFailed'
  | 'errors.authUnavailable'
  | 'errors.forbidden'
  | 'errors.networkError'
  | 'errors.rateLimited'
  | 'errors.validationFailed'
>;

const fallbackKeyByAction = {
  login: 'auth.loginFailed',
  logout: 'auth.logoutFailed',
  register: 'auth.registerFailed',
} as const satisfies Record<AuthAction, AuthErrorKey>;

export function resolveAuthErrorKey(error: unknown, action: AuthAction): AuthErrorKey {
  const errorCode = getAuthErrorCode(error);
  const status = getAuthErrorStatus(error);

  if (
    action === 'login' &&
    (errorCode === ApiErrorCode.AUTH_INVALID_CREDENTIALS ||
      errorCode === ApiErrorCode.AUTH_UNAUTHORIZED)
  ) {
    return 'auth.invalidCredentials';
  }

  if (action === 'register' && errorCode === ApiErrorCode.USER_EMAIL_ALREADY_EXISTS) {
    return 'auth.emailAlreadyExists';
  }

  if (errorCode === ApiErrorCode.AUTH_ACCOUNT_DISABLED) {
    return 'auth.accountDisabled';
  }

  if (errorCode === ApiErrorCode.AUTH_FORBIDDEN || status === 403) {
    return 'errors.forbidden';
  }

  if (action === 'login' && status === 401) {
    return 'auth.invalidCredentials';
  }

  if (
    errorCode === ApiErrorCode.COMMON_INVALID_INPUT ||
    errorCode === ApiErrorCode.COMMON_VALIDATION_FAILED ||
    status === 400 ||
    status === 422
  ) {
    return 'errors.validationFailed';
  }

  if (errorCode === ApiErrorCode.COMMON_RATE_LIMITED || status === 429) {
    return 'errors.rateLimited';
  }

  if (errorCode === ClientErrorCode.NETWORK_ERROR) {
    return 'errors.networkError';
  }

  if (
    errorCode === ClientErrorCode.FETCH_ERROR ||
    errorCode === ClientErrorCode.INVALID_RESPONSE ||
    errorCode === ClientErrorCode.TIMEOUT ||
    errorCode === ClientErrorCode.UNKNOWN ||
    errorCode === ApiErrorCode.COMMON_INTERNAL ||
    errorCode === ApiErrorCode.COMMON_SERVICE_UNAVAILABLE ||
    errorCode === ApiErrorCode.COMMON_TIMEOUT ||
    (status !== undefined && status >= 500)
  ) {
    return 'errors.authUnavailable';
  }

  return fallbackKeyByAction[action];
}
