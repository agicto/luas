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
export async function getT<P extends AllScopePaths>(
  scope: P
): Promise<ScopedTranslations<P>>;
export async function getT(
  scope?: AllScopePaths
): Promise<UnifiedTranslations | ScopedTranslations<AllScopePaths>> {
  const rootT = (await getTranslations()) as unknown as BaseTranslator;
  return scope ? createTranslator(rootT, scope) : createTranslator(rootT);
}
