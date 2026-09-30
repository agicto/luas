import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';

/** Maps a failed setting change to its i18n key; upstream messages are never shown. */
export function settingErrorKey(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return 'settings.errors.unavailable';
  }
  switch (error.errorCode) {
    case ApiErrorCode.SETTING_VERSION_CONFLICT:
      return 'settings.errors.conflict';
    case ApiErrorCode.SETTING_INVALID_VALUE:
      return 'settings.errors.invalid';
    default:
      return 'settings.errors.unavailable';
  }
}

export function isInactiveStarter(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404;
}
