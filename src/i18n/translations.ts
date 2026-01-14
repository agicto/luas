'use client';

import { useTranslations } from 'next-intl';
import { getTranslations } from 'next-intl/server';
import type { Messages } from './modules';

// ============================================================================
// Unified Translation Functions
// ============================================================================

/**
 * Namespace translator types for each module.
 */
type CommonTranslator = ReturnType<typeof useTranslations<'common'>>;
type AuthTranslator = ReturnType<typeof useTranslations<'auth'>>;
type NavTranslator = ReturnType<typeof useTranslations<'nav'>>;
type SettingsTranslator = ReturnType<typeof useTranslations<'settings'>>;
type ErrorsTranslator = ReturnType<typeof useTranslations<'errors'>>;
type MetadataTranslator = ReturnType<typeof useTranslations<'metadata'>>;

// ============================================================================
// Dot Notation Type Utilities
// ============================================================================

/**
 * Generate all valid dot-notation keys from the Messages type.
 * This creates a union like 'common.save' | 'common.loading' | 'auth.login' | ...
 */
type DotNotationKeys<T, Prefix extends string = ''> = T extends object
  ? {
      [K in keyof T]: K extends string
        ? T[K] extends object
          ? DotNotationKeys<T[K], `${Prefix}${K}.`>
          : `${Prefix}${K}`
        : never;
    }[keyof T]
  : never;

/**
 * All valid translation keys in dot notation format.
 * Example: 'common.save' | 'common.loading' | 'auth.login' | ...
 */
type AllTranslationKeys = DotNotationKeys<Messages>;

/**
 * Dot notation translator function type.
 * Provides strong typing for t('module.key') style calls.
 */
type DotNotationTranslator = (
  key: AllTranslationKeys,
  values?: Record<string, string | number | Date>,
) => string;

/**
 * Unified translation type.
 * 
 * This is a hybrid type that:
 * 1. Can be called as a function with dot notation: `t('common.save')`
 * 2. Has namespace properties for direct access: `t.common('save')`
 * 
 * Both usages are fully type-safe with IDE auto-completion.
 */
export type UnifiedTranslations = DotNotationTranslator & {
  common: CommonTranslator;
  auth: AuthTranslator;
  nav: NavTranslator;
  settings: SettingsTranslator;
  errors: ErrorsTranslator;
  metadata: MetadataTranslator;
};

/**
 * Universal client-side translation hook.
 * 
 * Supports both dot notation and namespace-based access:
 * - t('common.save') - dot notation
 * - t.common('save') - namespace-based (backward compatible)
 */
export function useT(): UnifiedTranslations {
  const rootT = useTranslations();
  
  const t = ((key: AllTranslationKeys, values?: Record<string, string | number | Date>) => {
    return rootT(key as Parameters<typeof rootT>[0], values);
  }) as UnifiedTranslations;
  
  t.common = useTranslations('common');
  t.auth = useTranslations('auth');
  t.nav = useTranslations('nav');
  t.settings = useTranslations('settings');
  t.errors = useTranslations('errors');
  t.metadata = useTranslations('metadata');
  
  return t;
}

/**
 * Universal server-side translation function.
 * 
 * Supports both dot notation and namespace-based access:
 * - t('common.save') - dot notation
 * - t.common('save') - namespace-based (backward compatible)
 */
export async function getT(): Promise<UnifiedTranslations> {
  const [rootT, common, auth, nav, settings, errors, metadata] = await Promise.all([
    getTranslations(),
    getTranslations('common'),
    getTranslations('auth'),
    getTranslations('nav'),
    getTranslations('settings'),
    getTranslations('errors'),
    getTranslations('metadata'),
  ]);

  const t = ((key: AllTranslationKeys, values?: Record<string, string | number | Date>) => {
    return rootT(key as Parameters<typeof rootT>[0], values);
  }) as UnifiedTranslations;
  
  t.common = common;
  t.auth = auth;
  t.nav = nav;
  t.settings = settings;
  t.errors = errors;
  t.metadata = metadata;
  
  return t;
}
