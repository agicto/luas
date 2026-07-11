export const AVAILABLE_MODULES = [
  'common',
  'auth',
  'nav',
  'settings',
  'errors',
  'metadata',
  'dashboard',
  'test',
] as const;

export type ModuleName = (typeof AVAILABLE_MODULES)[number];
