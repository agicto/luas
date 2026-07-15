import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

export type OrganizationErrorKey =
  | 'errors.forbidden'
  | 'errors.generic'
  | 'errors.invalidResponse'
  | 'errors.notFound'
  | 'errors.slugConflict'
  | 'errors.unavailable';

export function resolveOrganizationErrorKey(error: unknown): OrganizationErrorKey {
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
  if (error.errorCode === ApiErrorCode.ORGANIZATION_NOT_FOUND) {
    return 'errors.notFound';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_SLUG_ALREADY_EXISTS) {
    return 'errors.slugConflict';
  }
  return 'errors.generic';
}

export function hasOrganizationFieldError(error: unknown, field: string): boolean {
  return error instanceof ApiError && Boolean(error.fieldErrors?.[field]?.length);
}
