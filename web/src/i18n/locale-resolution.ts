import { locales, localeMapping, type Locale } from './locales';

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
    .map((entry, order) => {
      const [tag, ...parameters] = entry.split(';');
      const qualityParameter = parameters.find(parameter => parameter.trim().startsWith('q='));
      const quality = qualityParameter ? Number.parseFloat(qualityParameter.trim().slice(2)) : 1;

      return { tag: tag?.trim().toLowerCase(), quality, order };
    })
    .filter(
      (entry): entry is { tag: string; quality: number; order: number } =>
        Boolean(entry.tag) && Number.isFinite(entry.quality) && entry.quality > 0
    )
    .sort((left, right) => right.quality - left.quality || left.order - right.order);

  for (const { tag } of languages) {
    const exactMatch = localeMapping[tag];

    if (exactMatch) {
      return exactMatch;
    }

    const languageOnly = tag.split('-')[0];
    const languageMatch = localeMapping[languageOnly];

    if (languageMatch) {
      return languageMatch;
    }
  }

  return undefined;
}
