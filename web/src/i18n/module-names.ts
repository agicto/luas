export const AVAILABLE_MODULES = [
  'common',
  'auth',
  'nav',
  'site',
  'console',
  'organization',
  'permission',
  'notification',
  'settings',
  'errors',
  'metadata',
  'test',
] as const;

export type ModuleName = (typeof AVAILABLE_MODULES)[number];
