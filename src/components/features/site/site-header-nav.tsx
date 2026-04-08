/**
 * @component SiteHeaderNav
 * @category Feature
 * @status Stable
 * @description The main navigation component for the public site header, supporting localized links and active state detection.
 * @usage Place in the site header for global navigation.
 * @example
 * <SiteHeaderNav />
 */
'use client';

import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { ROUTES } from '@/constants/routes';
import { useAuthStore } from '@/features/auth/store/auth-store';
import { useT } from '@/i18n';

/**
 * Site Header Navigation Component
 * 
 * Shows different navigation options based on auth state.
 * When logged in: shows Dashboard link instead of Sign In/Get Started.
 * When not logged in: shows Sign In and Get Started buttons.
 */
export function SiteHeaderNav() {
  const t = useT();
  const isAuthenticated = useAuthStore.use.isAuthenticated();
  const user = useAuthStore.use.user();
  const isSystemReady = useAuthStore.use.isSystemReady();

  // During initialization, show sign-in buttons (safe default)
  if (!isSystemReady) {
    return (
      <div className="flex items-center gap-4">
        <Link href={ROUTES.AUTH.LOGIN}>
          <Button variant="ghost" size="sm">{t.auth('signIn')}</Button>
        </Link>
        <Link href={ROUTES.AUTH.REGISTER}>
          <Button size="sm">{t.auth('getStarted')}</Button>
        </Link>
      </div>
    );
  }

  if (isAuthenticated && user) {
    return (
      <div className="flex items-center gap-4">
        <Link href={ROUTES.CONSOLE.HOME}>
          <Button variant="ghost" size="sm">{t('nav.dashboard')}</Button>
        </Link>
        <Link href={ROUTES.CONSOLE.SETTINGS}>
          <Button size="sm" variant="outline">{user.name || t('nav.profile')}</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-4">
      <Link href={ROUTES.AUTH.LOGIN}>
        <Button variant="ghost" size="sm">{t.auth('signIn')}</Button>
      </Link>
      <Link href={ROUTES.AUTH.REGISTER}>
        <Button size="sm">{t.auth('getStarted')}</Button>
      </Link>
    </div>
  );
}
