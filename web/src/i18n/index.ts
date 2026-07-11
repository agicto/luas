// i18n barrel export
export { 
  locales, 
  defaultLocale, 
  localeNames, 
  localeMapping,
  isLocaleSwitcherEnabled,
  type Locale 
} from './config';

export { messages, type Messages } from './modules';
export { loadAllModules, loadModule } from './loader';

// Translation hooks/functions
export { 
  useT,
  type AllScopePaths,
  type AllTranslationKeys,
  type ScopedTranslationKeys,
  type ScopedTranslations,
  type Translators,
  type UnifiedTranslations,
} from './translations';
