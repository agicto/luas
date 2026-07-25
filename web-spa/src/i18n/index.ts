import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { env } from '@/config/env';
import { resources, type SupportedLocale } from './resources';

const localeStorageKey = 'luas-spa-locale';
const supportedLocales = Object.keys(resources) as SupportedLocale[];

function isSupportedLocale(value: string | null | undefined): value is SupportedLocale {
  return supportedLocales.includes(value as SupportedLocale);
}

function resolveInitialLocale(): SupportedLocale {
  const stored = window.localStorage.getItem(localeStorageKey);
  if (isSupportedLocale(stored)) {
    return stored;
  }

  const browserLocale = window.navigator.language;
  if (isSupportedLocale(browserLocale)) {
    return browserLocale;
  }
  if (browserLocale.toLowerCase().startsWith('zh')) {
    return 'zh-CN';
  }
  return env.DEFAULT_LOCALE;
}

void i18n.use(initReactI18next).init({
  resources,
  lng: resolveInitialLocale(),
  fallbackLng: 'en',
  supportedLngs: supportedLocales,
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
});

document.documentElement.lang = i18n.language;

i18n.on('languageChanged', (locale) => {
  if (isSupportedLocale(locale)) {
    window.localStorage.setItem(localeStorageKey, locale);
    document.documentElement.lang = locale;
  }
});

export async function changeLocale(locale: SupportedLocale) {
  await i18n.changeLanguage(locale);
}

export { i18n };
