// Errors translations - English (US)
const messages = {
  notFound: 'Page not found',
  serverError: 'Server error',
  networkError: 'Network error, please check your connection',
  rateLimited: 'Too many attempts, please try again later',
  unauthorized: 'Please login to continue',
  validationFailed: 'Please review the highlighted fields',
  forbidden: "You don't have permission to access this resource",
  authForbiddenDescription:
    'This session cannot access the protected application. Contact an administrator if this is unexpected.',
  authUnavailable: 'Unable to verify your session',
  authUnavailableDescription: 'Your session was not changed. Check your connection and try again.',
} as const;

export default messages;
export type ErrorsMessages = typeof messages;
