import { type Locale } from './config';
import { AVAILABLE_MODULES, type ModuleName } from './module-names';
import type { Messages as StrictMessages } from './modules';

export { AVAILABLE_MODULES, type ModuleName } from './module-names';

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
  site: {
    'zh-Hans': () => import('./modules/site/zh-Hans'),
    'en-US': () => import('./modules/site/en-US'),
  },
  console: {
    'zh-Hans': () => import('./modules/console/zh-Hans'),
    'en-US': () => import('./modules/console/en-US'),
  },
  organization: {
    'zh-Hans': () => import('./modules/organization/zh-Hans'),
    'en-US': () => import('./modules/organization/en-US'),
  },
  permission: {
    'zh-Hans': () => import('./modules/permission/zh-Hans'),
    'en-US': () => import('./modules/permission/en-US'),
  },
  notification: {
    'zh-Hans': () => import('./modules/notification/zh-Hans'),
    'en-US': () => import('./modules/notification/en-US'),
  },
  asset: {
    'zh-Hans': () => import('./modules/asset/zh-Hans'),
    'en-US': () => import('./modules/asset/en-US'),
  },
  setting: {
    'zh-Hans': () => import('./modules/setting/zh-Hans'),
    'en-US': () => import('./modules/setting/en-US'),
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
  const entries = await Promise.all(
    AVAILABLE_MODULES.map(async moduleName => {
      try {
        const loadedModule = await moduleRegistry[moduleName][locale]();
        return [moduleName, unwrapModule(loadedModule)] as const;
      } catch (error) {
        console.warn(`Failed to load ${moduleName} module for locale ${locale}:`, error);
        return [moduleName, {}] as const;
      }
    })
  );

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
