import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';

/** Maps a failed replay to its i18n key; upstream messages are never shown. */
export function replayErrorKey(error: unknown): string {
  if (!(error instanceof ApiError)) {
    return 'webhooks.errors.unavailable';
  }
  switch (error.errorCode) {
    case ApiErrorCode.WEBHOOK_REPLAY_NOT_ALLOWED:
      return 'webhooks.errors.replayNotAllowed';
    case ApiErrorCode.WEBHOOK_DELIVERY_NOT_FOUND:
      return 'webhooks.errors.deliveryNotFound';
    default:
      return 'webhooks.errors.unavailable';
  }
}
