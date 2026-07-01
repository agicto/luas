import { env } from '@/config/env';
import {
  locales,
  localeMapping,
  localeNames,
  type Locale,
} from './locales';

export { locales, localeMapping, localeNames, type Locale };

export const defaultLocale: Locale = env.NEXT_PUBLIC_DEFAULT_LOCALE;
export const isLocaleSwitcherEnabled = env.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED;
