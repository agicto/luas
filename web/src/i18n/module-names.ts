export const AVAILABLE_MODULES = [
  'common',
  'auth',
  'nav',
  'site',
  'console',
  'organization',
  'permission',
  'notification',
  'asset',
  'setting',
  'usage',
  'webhook',
  'settings',
  'errors',
  'metadata',
  'test',
  // fastbuild:slot web.i18n.modules
] as const;

export type ModuleName = (typeof AVAILABLE_MODULES)[number];
