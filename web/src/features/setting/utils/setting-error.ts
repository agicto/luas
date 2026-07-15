import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

export type SettingErrorKey =
  | 'errors.forbidden'
  | 'errors.generic'
  | 'errors.invalidResponse'
  | 'errors.invalidValue'
  | 'errors.notFound'
  | 'errors.preconditionRequired'
  | 'errors.unavailable'
  | 'errors.versionConflict';

export function resolveSettingErrorKey(error: unknown): SettingErrorKey {
  if (!(error instanceof ApiError)) return 'errors.generic';
  if (error.errorCode === ClientErrorCode.INVALID_RESPONSE) {
    return 'errors.invalidResponse';
  }
  if (
    error.errorCode === ClientErrorCode.NETWORK_ERROR ||
    error.errorCode === ClientErrorCode.TIMEOUT ||
    error.errorCode === ApiErrorCode.COMMON_TIMEOUT ||
    error.errorCode === ApiErrorCode.COMMON_SERVICE_UNAVAILABLE
  ) {
    return 'errors.unavailable';
  }
  if (
    error.errorCode === ApiErrorCode.AUTH_FORBIDDEN ||
    error.errorCode === ApiErrorCode.PERMISSION_DENIED
  ) {
    return 'errors.forbidden';
  }
  if (error.errorCode === ApiErrorCode.SETTING_NOT_FOUND) {
    return 'errors.notFound';
  }
  if (error.errorCode === ApiErrorCode.SETTING_INVALID_VALUE) {
    return 'errors.invalidValue';
  }
  if (error.errorCode === ApiErrorCode.SETTING_VERSION_CONFLICT) {
    return 'errors.versionConflict';
  }
  if (error.errorCode === ApiErrorCode.SETTING_PRECONDITION_REQUIRED) {
    return 'errors.preconditionRequired';
  }
  return 'errors.generic';
}
