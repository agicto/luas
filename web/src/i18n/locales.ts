export const locales = ['zh-Hans', 'en-US'] as const;
export type Locale = (typeof locales)[number];

export const defaultLocaleFallback: Locale = 'zh-Hans';

export const localeNames: Record<Locale, string> = {
  'zh-Hans': '简体中文',
  'en-US': 'English',
};

export const localeMapping: Record<string, Locale> = {
  zh: 'zh-Hans',
  'zh-cn': 'zh-Hans',
  'zh-hans': 'zh-Hans',
  en: 'en-US',
  'en-us': 'en-US',
};
