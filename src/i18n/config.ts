// i18n configuration

import { publicEnv } from '@/config/env';

// Supported locales (BCP 47 format)
export const locales = ['zh-Hans', 'en-US'] as const;
export type Locale = (typeof locales)[number];

// Default locale from environment variable or fallback to zh-Hans
export const defaultLocale: Locale = publicEnv.NEXT_PUBLIC_DEFAULT_LOCALE as Locale;

// Whether locale switcher is enabled (can be disabled via environment variable)
export const isLocaleSwitcherEnabled = publicEnv.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED;


// Locale display names
export const localeNames: Record<Locale, string> = {
  'zh-Hans': '简体中文',
  'en-US': 'English (US)',
};

// Locale to Accept-Language header mapping for browser detection
export const localeMapping: Record<string, Locale> = {
  'zh': 'zh-Hans',
  'zh-CN': 'zh-Hans',
  'zh-Hans': 'zh-Hans',
  'en': 'en-US',
  'en-US': 'en-US',
};
