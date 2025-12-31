// i18n configuration

// Supported locales (BCP 47 format)
export const locales = ['zh-Hans', 'en-US'] as const;
export type Locale = (typeof locales)[number];

// Default locale from environment variable or fallback to zh-Hans
export const defaultLocale: Locale = 
  (process.env.NEXT_PUBLIC_DEFAULT_LOCALE as Locale) || 'zh-Hans';

// Whether locale switcher is enabled (can be disabled via environment variable)
export const isLocaleSwitcherEnabled = 
  process.env.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED !== 'false';

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
