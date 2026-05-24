import { type Locale } from './config';
import type { Messages as StrictMessages } from './modules';

// Define available modules
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

// Interface for loaded messages
export type Messages = StrictMessages;

// Translation module loader with type safety
type MessageNamespace = Record<string, unknown>;
type ModuleLoader = () => Promise<{ default: MessageNamespace } | MessageNamespace>;
type ModuleRegistry = {
  [K in ModuleName]: Record<Locale, ModuleLoader>;
};

const moduleRegistry: ModuleRegistry = {
  common: {
    'zh-Hans': () => import('./modules/common/zh-Hans'),
    'en-US': () => import('./modules/common/en-US'),
  },
  auth: {
    'zh-Hans': () => import('./modules/auth/zh-Hans'),
    'en-US': () => import('./modules/auth/en-US'),
  },
  nav: {
    'zh-Hans': () => import('./modules/nav/zh-Hans'),
    'en-US': () => import('./modules/nav/en-US'),
  },
  settings: {
    'zh-Hans': () => import('./modules/settings/zh-Hans'),
    'en-US': () => import('./modules/settings/en-US'),
  },
  errors: {
    'zh-Hans': () => import('./modules/errors/zh-Hans'),
    'en-US': () => import('./modules/errors/en-US'),
  },
  metadata: {
    'zh-Hans': () => import('./modules/metadata/zh-Hans'),
    'en-US': () => import('./modules/metadata/en-US'),
  },
  dashboard: {
    'zh-Hans': () => import('./modules/dashboard/zh-Hans'),
    'en-US': () => import('./modules/dashboard/en-US'),
  },
  test: {
    'zh-Hans': () => import('./modules/test/zh-Hans'),
    'en-US': () => import('./modules/test/en-US'),
  },
};

type ModuleKey = keyof typeof moduleRegistry;

function unwrapModule(moduleData: Awaited<ReturnType<ModuleLoader>>): MessageNamespace {
  if ('default' in moduleData) {
    return moduleData.default as MessageNamespace;
  }

  return moduleData as MessageNamespace;
}

/**
 * Load all translation modules for a given locale
 */
export async function loadAllModules(locale: Locale): Promise<Messages> {
  const entries = await Promise.all(AVAILABLE_MODULES.map(async (moduleName) => {
    try {
      const loadedModule = await moduleRegistry[moduleName][locale]();
      return [moduleName, unwrapModule(loadedModule)] as const;
    } catch (error) {
      console.warn(`Failed to load ${moduleName} module for locale ${locale}:`, error);
      return [moduleName, {}] as const;
    }
  }));

  return Object.fromEntries(entries) as Messages;
}

/**
 * Load a specific module for a given locale
 */
export async function loadModule<K extends ModuleKey>(
  moduleKey: K,
  locale: Locale
): Promise<Messages[K]> {
  try {
    const loadedModule = await moduleRegistry[moduleKey][locale]();
    return unwrapModule(loadedModule) as Messages[K];
  } catch (error) {
    console.warn(`Failed to load ${moduleKey} module for locale ${locale}:`, error);
    return {} as Messages[K];
  }
}
