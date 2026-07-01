import {
  locales,
  localeMapping,
  type Locale,
} from './locales';

interface ResolveRequestLocaleOptions {
  cookieLocale?: string | null;
  acceptLanguage?: string | null;
  defaultLocale: Locale;
}

export function isSupportedLocale(value: unknown): value is Locale {
  return typeof value === 'string' && locales.includes(value as Locale);
}

export function resolveRequestLocale({
  cookieLocale,
  acceptLanguage,
  defaultLocale,
}: ResolveRequestLocaleOptions): Locale {
  if (isSupportedLocale(cookieLocale)) {
    return cookieLocale;
  }

  return resolveAcceptLanguageLocale(acceptLanguage) ?? defaultLocale;
}

export function resolveAcceptLanguageLocale(acceptLanguage?: string | null): Locale | undefined {
  if (!acceptLanguage) {
    return undefined;
  }

  const languages = acceptLanguage
    .split(',')
    .map((entry) => entry.split(';')[0]?.trim())
    .filter((entry): entry is string => Boolean(entry));

  for (const language of languages) {
    const exactMatch = localeMapping[language];

    if (exactMatch) {
      return exactMatch;
    }

    const languageOnly = language.split('-')[0];
    const languageMatch = localeMapping[languageOnly];

    if (languageMatch) {
      return languageMatch;
    }
  }

  return undefined;
}
