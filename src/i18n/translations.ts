'use client';

import { useTranslations } from 'next-intl';
import { getTranslations } from 'next-intl/server';
import { AVAILABLE_MODULES } from './loader';
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
export type AllScopePaths = DotNotationKeys<Messages>; 

/**
 * Type utility to get the subtree of keys after a prefix.
 */
type ShiftingKeys<T, P extends string> = P extends `${infer Head}.${infer Tail}`
  ? Head extends keyof T ? ShiftingKeys<T[Head], Tail> : never
  : P extends keyof T ? DotNotationKeys<T[P]> : never;

/**
 * Unified translation type.
 */
export type UnifiedTranslations = {
  (key: AllTranslationKeys, values?: any): string;
} & Translators;

/**
 * Scoped translation type.
 */
export type ScopedTranslations<P extends string> = (
  key: ShiftingKeys<Messages, P>,
  values?: any
) => string;

/**
 * Universal client-side translation hook.
 */
export function useT(): UnifiedTranslations;
export function useT<P extends AllScopePaths>(scope: P): ScopedTranslations<P>;
export function useT(scope?: string): any {
  const rootT = useTranslations(scope as any);
  
  const h = (key: string, values?: any) => (rootT as any)(key, values);
  const t = h as any;
  
  if (!scope) {
    AVAILABLE_MODULES.forEach(module => {
      // @ts-ignore - dynamic assignment
      t[module] = useTranslations(module);
    });
  }
  
  return t;
}

/**
 * Universal server-side translation function.
 */
export async function getT(): Promise<UnifiedTranslations>;
export async function getT<P extends AllScopePaths>(scope: P): Promise<ScopedTranslations<P>>;
export async function getT(scope?: string): Promise<any> {
  const rootT = await getTranslations(scope as any);

  const h = (key: string, values?: any) => (rootT as any)(key, values);
  const t = h as any;
  
  if (!scope) {
    await Promise.all(
      AVAILABLE_MODULES.map(async module => {
         // @ts-ignore - dynamic assignment
        t[module] = await getTranslations(module);
      })
    );
  }
  
  return t;
}
