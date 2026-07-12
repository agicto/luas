import { describe, expect, expectTypeOf, it } from 'vitest';

import type {
  AllScopePaths,
  AllTranslationKeys,
  ScopedTranslations,
} from '@/i18n';
import {
  createTranslator,
  type BaseTranslator,
} from '@/i18n/translation-shared';

describe('i18n type contracts', () => {
  it('distinguishes object scopes from translatable leaf keys', () => {
    expectTypeOf<'settings'>().toMatchTypeOf<AllScopePaths>();
    expectTypeOf<'test.level1.level2'>().toMatchTypeOf<AllScopePaths>();
    expectTypeOf<'settings.title'>().not.toMatchTypeOf<AllScopePaths>();
    expectTypeOf<'missing.scope'>().not.toMatchTypeOf<AllScopePaths>();

    expectTypeOf<'settings.title'>().toMatchTypeOf<AllTranslationKeys>();
    expectTypeOf<'test.level1.level2.message'>().toMatchTypeOf<AllTranslationKeys>();
    expectTypeOf<'settings'>().not.toMatchTypeOf<AllTranslationKeys>();
    expectTypeOf<'test.level1.level2'>().not.toMatchTypeOf<AllTranslationKeys>();
  });

  it('derives relative leaf keys for a selected scope', () => {
    type SettingsSystemKey = Parameters<
      ScopedTranslations<'settings.system'>
    >[0];
    type TestLevel2Key = Parameters<
      ScopedTranslations<'test.level1.level2'>
    >[0];

    expectTypeOf<SettingsSystemKey>().toEqualTypeOf<
      | 'title'
      | 'description'
      | 'companyName'
      | 'companyPlaceholder'
      | 'companyDefault'
      | 'websiteUrl'
      | 'websitePlaceholder'
      | 'supportEmail'
      | 'supportEmailPlaceholder'
      | 'save'
    >();
    expectTypeOf<TestLevel2Key>().toEqualTypeOf<
      | 'title'
      | 'message'
      | 'level3.title'
      | 'level3.message'
      | 'level3.level4.title'
      | 'level3.level4.message'
      | 'level3.level4.deepValue'
    >();
  });

  it('preserves scoped runtime key composition', () => {
    const baseTranslator: BaseTranslator = (key) => key;
    const t = createTranslator(baseTranslator, 'test.level1.level2');

    expect(t('message')).toBe('test.level1.level2.message');
    expect(t('level3.level4.deepValue')).toBe(
      'test.level1.level2.level3.level4.deepValue'
    );
  });
});
