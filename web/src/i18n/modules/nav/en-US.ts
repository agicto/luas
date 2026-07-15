// Nav translations - English (US)
import type { NavMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  home: 'Home',
  console: 'Console',
  organizations: 'Organizations',
  settings: 'Settings',
  profile: 'Profile',
  analytics: 'Analytics',
  styleguide: 'Styleguide',
} as const satisfies LocaleMessageShape<NavMessages>;

export default messages;
