// i18n barrel export
export { 
  locales, 
  defaultLocale, 
  localeNames, 
  localeMapping,
  isLocaleSwitcherEnabled,
  type Locale 
} from './config';

export { getMessages, type Messages } from './modules';

// Translation hooks/functions
export { 
  useClientT, 
  useServerT, 
  type ClientTranslations, 
  type ServerTranslations 
} from './translations';
