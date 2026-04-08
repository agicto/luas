'use client';

import { useTranslations } from 'next-intl';
import { getTranslations } from 'next-intl/server';
import { AVAILABLE_MODULES, type ModuleName } from './loader';
import type { Messages } from './modules';

// ============================================================================
// Unified Translation Functions
// ============================================================================

/**
 * Namespace translator types for each module.
 */
export type Translators = {
  [K in keyof Messages]: (
    key: DotNotationKeys<Messages[K]>, 
    values?: Record<string, string | number | Date>
  ) => string;
};

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
          ? `${Prefix}${K}` | DotNotationKeys<T[K], `${Prefix}${K}.`>
          : `${Prefix}${K}`
        : never;
    }[keyof T]
  : never;

/**
 * All valid translation keys in dot notation format.
 */
export type AllTranslationKeys = DotNotationKeys<Messages>;

/**
 * All valid paths that can be used as a prefix scope.
 */
export type AllScopePaths = string;

/**
 * Unified translation type.
 */
export type UnifiedTranslations = {
  (key: AllTranslationKeys, values?: TranslationValues): string;
} & Translators;

/**
 * Scoped translation type.
 */
export type ScopedTranslations = (
  key: string,
  values?: TranslationValues
) => string;

type TranslationValues = Record<string, string | number | Date>;
type BaseTranslator = (key: string, values?: TranslationValues) => string;
type ScopedTranslatorMap = Record<ModuleName, (key: string, values?: TranslationValues) => string>;

function withScope(scope: string | undefined, key: string): string {
  return scope ? `${scope}.${key}` : key;
}

function createScopedTranslatorMap(rootT: BaseTranslator): ScopedTranslatorMap {
  return Object.fromEntries(
    AVAILABLE_MODULES.map((moduleName) => [
      moduleName,
      (key: string, values?: TranslationValues) => rootT(withScope(moduleName, key), values),
    ])
  ) as ScopedTranslatorMap;
}

/**
 * Universal client-side translation hook.
 */
export function useT(): UnifiedTranslations;
export function useT<P extends AllScopePaths>(scope: P): ScopedTranslations;
export function useT(scope?: string): UnifiedTranslations | ScopedTranslations {
  const rootT = useTranslations() as unknown as BaseTranslator;

  if (scope) {
    return ((key: string, values?: TranslationValues) =>
      rootT(withScope(scope, key), values)) as ScopedTranslations;
  }

  return Object.assign(
    ((key: string, values?: TranslationValues) => rootT(key, values)) as UnifiedTranslations,
    createScopedTranslatorMap(rootT)
  );
}

/**
 * Universal server-side translation function.
 */
export async function getT(): Promise<UnifiedTranslations>;
export async function getT<P extends AllScopePaths>(scope: P): Promise<ScopedTranslations>;
export async function getT(scope?: string): Promise<UnifiedTranslations | ScopedTranslations> {
  const rootT = (await getTranslations()) as unknown as BaseTranslator;

  if (scope) {
    return ((key: string, values?: TranslationValues) =>
      rootT(withScope(scope, key), values)) as ScopedTranslations;
  }

  return Object.assign(
    ((key: string, values?: TranslationValues) => rootT(key, values)) as UnifiedTranslations,
    createScopedTranslatorMap(rootT)
  );
}
