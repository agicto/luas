// Nav translations - English (US)
const messages = {
  home: 'Home',
  console: 'Console',
  organizations: 'Organizations',
  assets: 'Assets',
  usage: 'Usage',
  settings: 'Settings',
  profile: 'Profile',
  analytics: 'Analytics',
  styleguide: 'Styleguide',
} as const;

export default messages;
export type NavMessages = typeof messages;
