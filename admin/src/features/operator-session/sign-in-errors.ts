import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';

/** Maps a sign-in failure to its i18n key; upstream messages are never shown. */
export function signInErrorKey(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return 'auth.errors.unavailable';
  }
  switch (error.errorCode) {
    case ApiErrorCode.AUTH_INVALID_CREDENTIALS:
    case ApiErrorCode.COMMON_VALIDATION_FAILED:
      return 'auth.errors.invalidCredentials';
    case ApiErrorCode.OPERATOR_FORBIDDEN:
      return 'auth.errors.notOperator';
    case ApiErrorCode.COMMON_RATE_LIMITED:
      return 'auth.errors.rateLimited';
    case ApiErrorCode.OPERATOR_ORIGIN_REJECTED:
      return 'auth.errors.originRejected';
    default:
      return 'auth.errors.unavailable';
  }
}
