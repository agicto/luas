import { describe, expect, it } from 'vitest';

import { normalizeLocale, resolvePreferredLocale } from './locale';

describe('Admin locale resolution', () => {
  it('normalizes supported language and region variants', () => {
    expect(normalizeLocale('en-US')).toBe('en-US');
    expect(normalizeLocale('en-GB')).toBe('en-US');
    expect(normalizeLocale('zh-CN')).toBe('zh-Hans');
    expect(normalizeLocale('zh-Hans')).toBe('zh-Hans');
    expect(normalizeLocale('fr-FR')).toBeUndefined();
  });

  it('selects the first supported preference and then the configured fallback', () => {
    expect(resolvePreferredLocale(['fr-FR', 'zh-CN', 'en-US'], 'en-US')).toBe('zh-Hans');
    expect(resolvePreferredLocale(['fr-FR'], 'en-US')).toBe('en-US');
  });
});
