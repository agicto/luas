// Common translations - English (US)
import type { CommonMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  loading: 'Loading...',
  error: 'An error occurred',
  retry: 'Retry',
  retryLater: 'Please try again later',
  save: 'Save',
  cancel: 'Cancel',
  confirm: 'Confirm',
  delete: 'Delete',
  edit: 'Edit',
  create: 'Create',
  search: 'Search',
  noData: 'No data available',
  success: 'Success',
  failed: 'Failed',
  
  // User CRUD messages
  userCreateSuccess: 'User created successfully',
  userCreateFailed: 'Failed to create user',
  userUpdateSuccess: 'User updated successfully',
  userUpdateFailed: 'Failed to update user',
  userDeleteSuccess: 'User deleted successfully',
  userDeleteFailed: 'Failed to delete user',
  userCreated: 'User {name} has been created',
  userUpdated: 'User {name} has been updated',
  
  // Date Picker
  year: 'Year',
  month: 'Month',
  hour: 'Hour',
  minute: 'Minute',
  second: 'Second',
  now: 'Now',
  today: 'Today',
  datePlaceholder: 'Select date',
  chooseDate: 'Choose date',
  time: 'Time',
  selectedDate: 'Selected date: {date}',
  toggleLanguage: 'Toggle language',
  selectLanguage: 'Select language',
  toggleTheme: 'Toggle theme',
  themeLight: 'Light',
  themeDark: 'Dark',
  themeSystem: 'System',
} as const satisfies LocaleMessageShape<CommonMessages>;

export default messages;
