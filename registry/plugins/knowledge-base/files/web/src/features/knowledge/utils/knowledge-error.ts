import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

export type KnowledgeErrorKey =
  | 'errors.forbidden'
  | 'errors.generic'
  | 'errors.invalidArticle'
  | 'errors.invalidResponse'
  | 'errors.notFound'
  | 'errors.organizationUnavailable'
  | 'errors.preconditionRequired'
  | 'errors.tooLarge'
  | 'errors.unavailable'
  | 'errors.versionConflict';

/** Maps a failure to stable local copy; backend message text is never shown. */
export function resolveKnowledgeErrorKey(error: unknown): KnowledgeErrorKey {
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
  if (
    error.errorCode === ApiErrorCode.ORGANIZATION_NOT_FOUND ||
    error.errorCode === ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED ||
    error.errorCode === ApiErrorCode.ORGANIZATION_CONTEXT_INVALID
  ) {
    return 'errors.organizationUnavailable';
  }
  if (error.errorCode === ApiErrorCode.KNOWLEDGE_ARTICLE_NOT_FOUND) return 'errors.notFound';
  if (
    error.errorCode === ApiErrorCode.KNOWLEDGE_INVALID_ARTICLE ||
    error.errorCode === ApiErrorCode.COMMON_VALIDATION_FAILED
  ) {
    return 'errors.invalidArticle';
  }
  if (error.errorCode === ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT) return 'errors.versionConflict';
  if (error.errorCode === ApiErrorCode.KNOWLEDGE_PRECONDITION_REQUIRED) {
    return 'errors.preconditionRequired';
  }
  if (error.errorCode === ApiErrorCode.COMMON_REQUEST_TOO_LARGE) return 'errors.tooLarge';
  return 'errors.generic';
}
