'use client';

import { useTranslations } from 'next-intl';
import { getTranslations } from 'next-intl/server';

// ============================================================================
// Unified Translation Functions
// ============================================================================

/**
 * Unified translation object type.
 * Provides module-based access to all translation namespaces.
 */
export type UnifiedTranslations = {
  common: ReturnType<typeof useTranslations<'common'>>;
  auth: ReturnType<typeof useTranslations<'auth'>>;
  nav: ReturnType<typeof useTranslations<'nav'>>;
  settings: ReturnType<typeof useTranslations<'settings'>>;
  errors: ReturnType<typeof useTranslations<'errors'>>;
  metadata: ReturnType<typeof useTranslations<'metadata'>>;
};

/**
 * Universal client-side translation hook.
 * 
 * Provides module-based access to all translation namespaces.
 * 
 * @example
 * ```tsx
 * 'use client';
 * import { useT } from '@/i18n';
 * 
 * function MyComponent() {
 *   const t = useT();
 *   return (
 *     <div>
 *       <button>{t.common('save')}</button>
 *       <p>{t.auth('login')}</p>
 *       <span>{t.errors('networkError')}</span>
 *     </div>
 *   );
 * }
 * ```
 */
export function useT(): UnifiedTranslations {
  return {
    common: useTranslations('common'),
    auth: useTranslations('auth'),
    nav: useTranslations('nav'),
    settings: useTranslations('settings'),
    errors: useTranslations('errors'),
    metadata: useTranslations('metadata'),
  };
}

/**
 * Universal server-side translation function.
 * 
 * Provides module-based access to all translation namespaces.
 * 
 * @example
 * ```tsx
 * // app/page.tsx (Server Component)
 * import { getT } from '@/i18n';
 * 
 * export default async function Page() {
 *   const t = await getT();
 *   return (
 *     <div>
 *       <h1>{t.common('loading')}</h1>
 *       <p>{t.nav('home')}</p>
 *     </div>
 *   );
 * }
 * ```
 */
export async function getT(): Promise<UnifiedTranslations> {
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

