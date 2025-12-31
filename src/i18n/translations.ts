'use client';

import { useTranslations } from 'next-intl';
import { getTranslations } from 'next-intl/server';

// ============================================================================
// Client-side Translation Hook
// ============================================================================

/**
 * Client-side translation hook for Client Components.
 * Provides access to all i18n modules.
 * 
 * @example
 * ```tsx
 * 'use client';
 * import { useClientT } from '@/i18n';
 * 
 * function LoginForm() {
 *   const t = useClientT();
 *   return <button>{t.common('save')}</button>;
 * }
 * ```
 */
export function useClientT() {
  return {
    common: useTranslations('common'),
    auth: useTranslations('auth'),
    nav: useTranslations('nav'),
    settings: useTranslations('settings'),
    errors: useTranslations('errors'),
    metadata: useTranslations('metadata'),
  };
}

export type ClientTranslations = ReturnType<typeof useClientT>;

// ============================================================================
// Server-side Translation Function
// ============================================================================

/**
 * Server-side translation function for Server Components.
 * Provides access to all i18n modules.
 * 
 * @example
 * ```tsx
 * // app/page.tsx (Server Component)
 * import { useServerT } from '@/i18n';
 * 
 * export default async function Page() {
 *   const t = await useServerT();
 *   return <h1>{t.common('loading')}</h1>;
 * }
 * ```
 */
export async function useServerT() {
  const [common, auth, nav, settings, errors, metadata] = await Promise.all([
    getTranslations('common'),
    getTranslations('auth'),
    getTranslations('nav'),
    getTranslations('settings'),
    getTranslations('errors'),
    getTranslations('metadata'),
  ]);

  return {
    common,
    auth,
    nav,
    settings,
    errors,
    metadata,
  };
}

export type ServerTranslations = Awaited<ReturnType<typeof useServerT>>;
