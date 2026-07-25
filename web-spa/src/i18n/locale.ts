import { resources, type SupportedLocale } from './resources';

export const supportedLocales = Object.keys(resources) as SupportedLocale[];

export function isSupportedLocale(value: unknown): value is SupportedLocale {
  return typeof value === 'string' && supportedLocales.includes(value as SupportedLocale);
}

export function normalizeLocale(value: string | null | undefined): SupportedLocale | undefined {
  if (!value) {
    return undefined;
  }

  if (isSupportedLocale(value)) {
    return value;
  }

  const normalized = value.toLowerCase();
  if (normalized.startsWith('zh')) {
    return 'zh-Hans';
  }
  if (normalized.startsWith('en')) {
    return 'en-US';
  }

  return undefined;
}

export function resolvePreferredLocale(
  candidates: readonly (string | null | undefined)[],
  fallback: SupportedLocale,
): SupportedLocale {
  for (const candidate of candidates) {
    const locale = normalizeLocale(candidate);
    if (locale) {
      return locale;
    }
  }

  return fallback;
}
