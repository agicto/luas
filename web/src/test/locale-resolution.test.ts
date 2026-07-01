import { describe, expect, it } from 'vitest';

import {
  isSupportedLocale,
  resolveAcceptLanguageLocale,
  resolveRequestLocale,
} from '@/i18n/locale-resolution';

describe('locale resolution', () => {
  it('accepts only configured locales', () => {
    expect(isSupportedLocale('zh-Hans')).toBe(true);
    expect(isSupportedLocale('en-US')).toBe(true);
    expect(isSupportedLocale('fr-FR')).toBe(false);
    expect(isSupportedLocale(undefined)).toBe(false);
  });

  it('prefers a supported locale cookie over the accept-language header', () => {
    expect(
      resolveRequestLocale({
        cookieLocale: 'en-US',
        acceptLanguage: 'zh-CN,zh;q=0.9',
        defaultLocale: 'zh-Hans',
      })
    ).toBe('en-US');
  });

  it('falls back to accept-language exact and language-only matches', () => {
    expect(resolveAcceptLanguageLocale('en-US,en;q=0.9')).toBe('en-US');
    expect(resolveAcceptLanguageLocale('zh-CN,zh;q=0.9')).toBe('zh-Hans');
    expect(resolveAcceptLanguageLocale('en-GB,en;q=0.9')).toBe('en-US');
  });

  it('uses the configured default when no request locale matches', () => {
    expect(
      resolveRequestLocale({
        cookieLocale: 'fr-FR',
        acceptLanguage: 'fr-FR,fr;q=0.9',
        defaultLocale: 'zh-Hans',
      })
    ).toBe('zh-Hans');
  });
});
