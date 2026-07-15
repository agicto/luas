import type { LocaleMessageShape } from '../../locale-message-shape';
import type { NotificationMessages } from './zh-Hans';

const messages = {
  open: 'Open notification center',
  title: 'Notifications',
  unreadCount: '{count} unread notifications',
  markAllRead: 'Mark all as read',
  empty: 'No notifications yet',
  loading: 'Loading notifications',
  loadError: 'Notifications could not be loaded. Try again.',
  retry: 'Retry',
  unread: 'Unread',
  preferences: 'Notification preferences',
  preferencesDescription: 'These settings affect future non-required notifications only.',
  inApp: 'In-app notifications',
  inAppDescription: 'Show new messages in the Luas console notification center.',
  email: 'Email notifications',
  emailDescription: 'Send new messages through the configured email provider.',
  cancel: 'Cancel',
  save: 'Save preferences',
  saved: 'Notification preferences updated',
  saveError: 'Notification preferences could not be saved.',
  readError: 'The notification read state could not be updated.',
  markAllError: 'Notifications could not be marked as read.',
} as const satisfies LocaleMessageShape<NotificationMessages>;

export default messages;
