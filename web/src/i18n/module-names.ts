export const AVAILABLE_MODULES = [
  'common',
  'auth',
  'nav',
  'site',
  'console',
  'organization',
  'settings',
  'errors',
  'metadata',
  'test',
] as const;

export type ModuleName = (typeof AVAILABLE_MODULES)[number];
