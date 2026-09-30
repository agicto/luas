import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';

/** Maps a failed user action to its i18n key; upstream messages are never shown. */
export function userActionErrorKey(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return 'users.errors.unavailable';
  }
  switch (error.errorCode) {
    case ApiErrorCode.OPERATOR_TARGET_PROTECTED:
      return 'users.errors.protected';
    case ApiErrorCode.USER_NOT_FOUND:
      return 'users.errors.notFound';
    default:
      return 'users.errors.unavailable';
  }
}
