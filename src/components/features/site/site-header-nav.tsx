'use client';

import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/store/auth-store';

/**
 * Site Header Navigation Component
 * 
 * Shows different navigation options based on auth state.
 * When logged in: shows Dashboard link instead of Sign In/Get Started.
 * When not logged in: shows Sign In and Get Started buttons.
 */
export function SiteHeaderNav() {
  const isAuthenticated = useAuthStore.use.isAuthenticated();
  const user = useAuthStore.use.user();
  const isSystemReady = useAuthStore.use.isSystemReady();

  // During initialization, show sign-in buttons (safe default)
  if (!isSystemReady) {
    return (
      <div className="flex items-center gap-4">
        <Link href="/login">
          <Button variant="ghost" size="sm">Sign In</Button>
        </Link>
        <Link href="/register">
          <Button size="sm">Get Started</Button>
        </Link>
      </div>
    );
  }

  if (isAuthenticated && user) {
    return (
      <div className="flex items-center gap-4">
        <Link href="/console">
          <Button variant="ghost" size="sm">Dashboard</Button>
        </Link>
        <Link href="/console/settings">
          <Button size="sm" variant="outline">{user.name || 'Profile'}</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-4">
      <Link href="/login">
        <Button variant="ghost" size="sm">Sign In</Button>
      </Link>
      <Link href="/register">
        <Button size="sm">Get Started</Button>
      </Link>
    </div>
  );
}
