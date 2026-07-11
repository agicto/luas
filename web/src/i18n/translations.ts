'use client';

import { useTranslations } from 'next-intl';
import {
  createTranslator,
  type AllScopePaths,
  type BaseTranslator,
  type ScopedTranslations,
  type UnifiedTranslations,
} from './translation-shared';

export type {
  AllScopePaths,
  AllTranslationKeys,
  ScopedTranslationKeys,
  ScopedTranslations,
  Translators,
  UnifiedTranslations,
} from './translation-shared';

/**
 * Universal client-side translation hook.
 */
export function useT(): UnifiedTranslations;
export function useT<P extends AllScopePaths>(scope: P): ScopedTranslations<P>;
export function useT(
  scope?: AllScopePaths
): UnifiedTranslations | ScopedTranslations<AllScopePaths> {
  const rootT = useTranslations() as unknown as BaseTranslator;
  return scope ? createTranslator(rootT, scope) : createTranslator(rootT);
}
