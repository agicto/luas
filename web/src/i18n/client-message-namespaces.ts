import type { ModuleName } from './module-names';

// Global namespaces are always serialized; route scopes add only what their client leaves use.
export const CLIENT_MESSAGE_NAMESPACES = {
  global: ['common', 'errors'],
  auth: ['auth'],
  console: ['auth', 'nav', 'console'],
  settings: ['settings'],
  organization: ['organization'],
  i18nTest: ['test'],
} as const satisfies Record<string, readonly ModuleName[]>;
