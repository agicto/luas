import { describe, expect, expectTypeOf, it } from 'vitest';

import type {
  AllScopePaths,
  AllTranslationKeys,
  LocaleMessageSchemaCoverageCheck,
  LocaleMessageVariableParityCheck,
  ScopedTranslations,
  TranslationValue,
  TranslationVariables,
} from '@/i18n';
import { createTranslator, type BaseTranslator } from '@/i18n/translation-shared';
import type { LocaleMessageVariableParity } from '@/i18n/locale-message-shape';

describe('i18n type contracts', () => {
  it('keeps locale ICU variables aligned with the base locale', () => {
    expectTypeOf<LocaleMessageSchemaCoverageCheck>().toEqualTypeOf<true>();
    expectTypeOf<LocaleMessageVariableParityCheck>().toEqualTypeOf<true>();
    expectTypeOf<
      LocaleMessageVariableParity<{ greeting: 'Hello {name}' }, { greeting: 'Welcome {user}' }>
    >().toEqualTypeOf<false>();
    expectTypeOf<
      LocaleMessageVariableParity<
        { summary: '{count, plural, =0 {None} other {# items}}' },
        { summary: '{total, plural, =0 {None} other {# items}}' }
      >
    >().toEqualTypeOf<false>();
  });

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
    type SettingErrorKey = Parameters<ScopedTranslations<'setting.errors'>>[0];
    type TestLevel2Key = Parameters<ScopedTranslations<'test.level1.level2'>>[0];

    expectTypeOf<SettingErrorKey>().toEqualTypeOf<
      | 'forbidden'
      | 'generic'
      | 'invalidResponse'
      | 'invalidValue'
      | 'notFound'
      | 'preconditionRequired'
      | 'unavailable'
      | 'versionConflict'
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

  it('derives ICU variables from each message literal', () => {
    expectTypeOf<TranslationVariables<'console.greeting.hello'>>().toEqualTypeOf<{
      name: TranslationValue;
    }>();
    expectTypeOf<TranslationVariables<'site.footer.copyright'>>().toEqualTypeOf<{
      year: TranslationValue;
    }>();
    expectTypeOf<
      TranslationVariables<'test.level1.level2.level3.level4.deepValue'>
    >().toEqualTypeOf<{ value: TranslationValue }>();
    expectTypeOf<TranslationVariables<'common.save'>>().toBeNever();
  });

  it('requires exact variables at global, namespace, and scoped call sites', () => {
    const baseTranslator: BaseTranslator = key => key;
    const globalT = createTranslator(baseTranslator);
    const scopedT = createTranslator(baseTranslator, 'console.greeting');
    const valuesWithExtraProperty = { name: 'Ada', tenant: 'Luas' };

    globalT('console.greeting.hello', { name: 'Ada' });
    globalT.auth('welcomeBackUser', { name: 'Ada' });
    scopedT('hello', { name: 'Ada' });

    if (false) {
      // @ts-expect-error - ICU variables are required.
      globalT('console.greeting.hello');
      // @ts-expect-error - Variable names come from the message literal.
      globalT('console.greeting.hello', { user: 'Ada' });
      // @ts-expect-error - Messages without variables reject a values object.
      globalT('common.save', { name: 'Ada' });
      // @ts-expect-error - Variables stored before the call still reject extras.
      globalT('console.greeting.hello', valuesWithExtraProperty);
      // @ts-expect-error - Scoped translators enforce the same variable contract.
      scopedT('hello');
    }
  });

  it('preserves scoped runtime key composition', () => {
    const baseTranslator: BaseTranslator = key => key;
    const t = createTranslator(baseTranslator, 'test.level1.level2');

    expect(t('message')).toBe('test.level1.level2.message');
    expect(t('level3.level4.deepValue', { value: 'test' })).toBe(
      'test.level1.level2.level3.level4.deepValue'
    );
  });
});
