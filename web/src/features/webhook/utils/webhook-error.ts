import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

export type WebhookErrorKey =
  | 'errors.deliveryNotFound'
  | 'errors.endpointNotFound'
  | 'errors.forbidden'
  | 'errors.generic'
  | 'errors.invalidEventType'
  | 'errors.invalidResponse'
  | 'errors.invalidTarget'
  | 'errors.preconditionRequired'
  | 'errors.replayNotAllowed'
  | 'errors.unavailable'
  | 'errors.versionConflict';

export function resolveWebhookErrorKey(error: unknown): WebhookErrorKey {
  if (!(error instanceof ApiError)) return 'errors.generic';
  if (error.errorCode === ClientErrorCode.INVALID_RESPONSE) return 'errors.invalidResponse';
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
  if (error.errorCode === ApiErrorCode.WEBHOOK_ENDPOINT_NOT_FOUND) {
    return 'errors.endpointNotFound';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_DELIVERY_NOT_FOUND) {
    return 'errors.deliveryNotFound';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_INVALID_TARGET) {
    return 'errors.invalidTarget';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_INVALID_EVENT_TYPE) {
    return 'errors.invalidEventType';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_ENDPOINT_VERSION_CONFLICT) {
    return 'errors.versionConflict';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_PRECONDITION_REQUIRED) {
    return 'errors.preconditionRequired';
  }
  if (error.errorCode === ApiErrorCode.WEBHOOK_REPLAY_NOT_ALLOWED) {
    return 'errors.replayNotAllowed';
  }
  return 'errors.generic';
}
