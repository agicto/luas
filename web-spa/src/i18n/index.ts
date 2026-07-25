import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { env } from '@/config/env';
import { resolvePreferredLocale, supportedLocales } from './locale';
import { resources, type SupportedLocale } from './resources';

const localeStorageKey = 'luas-spa-locale';

function resolveInitialLocale(): SupportedLocale {
  const stored = window.localStorage.getItem(localeStorageKey);
  const browserLocales = window.navigator.languages.length
    ? window.navigator.languages
    : [window.navigator.language];
  return resolvePreferredLocale([stored, ...browserLocales], env.DEFAULT_LOCALE);
}

void i18n.use(initReactI18next).init({
  resources,
  lng: resolveInitialLocale(),
  fallbackLng: 'en-US',
  supportedLngs: supportedLocales,
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
});

document.documentElement.lang = i18n.language;

i18n.on('languageChanged', (locale) => {
  if (supportedLocales.includes(locale as SupportedLocale)) {
    window.localStorage.setItem(localeStorageKey, locale);
    document.documentElement.lang = locale;
  }
});

export async function changeLocale(locale: SupportedLocale) {
  await i18n.changeLanguage(locale);
}

export { i18n };
