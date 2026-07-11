import 'server-only';

import { getTranslations } from 'next-intl/server';
import {
  createTranslator,
  type AllScopePaths,
  type BaseTranslator,
  type ScopedTranslations,
  type UnifiedTranslations,
} from './translation-shared';

export async function getT(): Promise<UnifiedTranslations>;
export async function getT<P extends AllScopePaths>(scope: P): Promise<ScopedTranslations>;
export async function getT(
  scope?: string
): Promise<UnifiedTranslations | ScopedTranslations> {
  const rootT = (await getTranslations()) as unknown as BaseTranslator;
  return scope ? createTranslator(rootT, scope) : createTranslator(rootT);
}
