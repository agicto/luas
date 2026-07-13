'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { authConfig } from '@/config/auth';
import { useAuthStore } from '@/features/auth/store/auth-store';
import { useT } from '@/i18n';

interface AuthGuardProps {
  children: React.ReactNode;
  /**
   * URL to redirect to if not authenticated
   * @default '/login'
   */
  redirectTo?: string;
  /**
   * Show loading indicator while checking auth
   * @default true
   */
  showLoading?: boolean;
}

/**
 * AuthGuard - Client component that protects routes
 * 
 * Checks if user is authenticated and redirects to login if not.
 * Use this in layouts to protect entire route groups.
 * 
 * @example
 * ```tsx
 * // app/(protected)/layout.tsx
 * export default function ProtectedLayout({ children }) {
 *   return <AuthGuard>{children}</AuthGuard>;
 * }
 * ```
 */
export function AuthGuard({ 
  children, 
  redirectTo = authConfig.routes.login,
  showLoading = true,
}: AuthGuardProps) {
  const router = useRouter();
  const pathname = usePathname();
  
  const status = useAuthStore.use.status();
  const t = useT();

  useEffect(() => {
    // Wait for system to be ready before checking auth
    if (status === 'idle' || status === 'loading') return;
    
    // If not loading and not authenticated, redirect to login
    if (status === 'unauthenticated') {
      // Encode current path for redirect after login
      const returnUrl = encodeURIComponent(pathname);
      router.replace(`${redirectTo}?returnUrl=${returnUrl}`);
    }
  }, [pathname, redirectTo, router, status]);

  // Show loading while system is initializing or checking auth
  if (status === 'idle' || status === 'loading') {
    if (showLoading) {
      return (
        <main
          aria-busy="true"
          className="flex min-h-screen items-center justify-center"
        >
          <div className="flex flex-col items-center space-y-4">
            <div
              aria-hidden="true"
              className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"
            />
            <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
          </div>
        </main>
      );
    }
    return null;
  }

  // If not authenticated, don't render children (will redirect)
  if (status === 'unauthenticated') {
    return null;
  }

  return <>{children}</>;
}
