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
  // fastbuild:slot web.i18n.nav.en-us
} as const;

export default messages;
export type NavMessages = typeof messages;
