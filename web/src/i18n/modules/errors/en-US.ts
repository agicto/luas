// Errors translations - English (US)
import type { ErrorsMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  notFound: 'Page not found',
  serverError: 'Server error',
  networkError: 'Network error, please check your connection',
  unauthorized: 'Please login to continue',
  forbidden: "You don't have permission to access this resource",
} as const satisfies LocaleMessageShape<ErrorsMessages>;

export default messages;
