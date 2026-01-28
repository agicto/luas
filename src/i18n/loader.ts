import { type Locale } from './config';

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

import type { Messages as StrictMessages } from './modules';

// Interface for loaded messages
export type Messages = StrictMessages;

// Translation module loader with type safety
const moduleRegistry = {
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
} as const;

type ModuleKey = keyof typeof moduleRegistry;

/**
 * Load all translation modules for a given locale
 */
export async function loadAllModules(locale: Locale): Promise<Messages> {
  const messages: any = {};

  // Load all modules in parallel for better performance
  const modulePromises = Object.entries(moduleRegistry).map(async ([key, modules]) => {
    try {
      const module = await (modules as any)[locale]();
      return [key, module.default || module];
    } catch (error) {
      console.warn(`Failed to load ${key} module for locale ${locale}:`, error);
      return [key, {}];
    }
  });

  const loadedModules = await Promise.all(modulePromises);

  // Merge all modules into the messages object
  for (const [key, moduleData] of loadedModules) {
    messages[key as string] = moduleData;
  }

  return messages as Messages;
}

/**
 * Load a specific module for a given locale
 */
export async function loadModule(
  moduleKey: ModuleKey,
  locale: Locale
): Promise<Record<string, any>> {
  try {
    const module = await (moduleRegistry[moduleKey] as any)[locale]();
    return module.default || module;
  } catch (error) {
    console.warn(`Failed to load ${moduleKey} module for locale ${locale}:`, error);
    return {};
  }
}
