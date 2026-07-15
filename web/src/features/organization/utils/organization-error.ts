import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

export type OrganizationErrorKey =
  | 'errors.forbidden'
  | 'errors.generic'
  | 'errors.invitationAlreadyPending'
  | 'errors.invitationEmailMismatch'
  | 'errors.invitationExpired'
  | 'errors.invitationInvalid'
  | 'errors.invitationNotFound'
  | 'errors.invalidResponse'
  | 'errors.memberAlreadyExists'
  | 'errors.memberNotFound'
  | 'errors.notFound'
  | 'errors.ownershipTransferRequired'
  | 'errors.ownershipTransferTargetInvalid'
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
  if (error.errorCode === ApiErrorCode.ORGANIZATION_INVITATION_INVALID) {
    return 'errors.invitationInvalid';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_INVITATION_EXPIRED) {
    return 'errors.invitationExpired';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_INVITATION_EMAIL_MISMATCH) {
    return 'errors.invitationEmailMismatch';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_INVITATION_ALREADY_PENDING) {
    return 'errors.invitationAlreadyPending';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_INVITATION_NOT_FOUND) {
    return 'errors.invitationNotFound';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_MEMBER_ALREADY_EXISTS) {
    return 'errors.memberAlreadyExists';
  }
  if (error.errorCode === ApiErrorCode.ORGANIZATION_MEMBER_NOT_FOUND) {
    return 'errors.memberNotFound';
  }
  if (
    error.errorCode === ApiErrorCode.ORGANIZATION_OWNERSHIP_TRANSFER_REQUIRED
  ) {
    return 'errors.ownershipTransferRequired';
  }
  if (
    error.errorCode ===
    ApiErrorCode.ORGANIZATION_OWNERSHIP_TRANSFER_TARGET_INVALID
  ) {
    return 'errors.ownershipTransferTargetInvalid';
  }
  return 'errors.generic';
}

export function hasOrganizationFieldError(error: unknown, field: string): boolean {
  return error instanceof ApiError && Boolean(error.fieldErrors?.[field]?.length);
}
