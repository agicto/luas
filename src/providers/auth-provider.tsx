'use client';

import { useEffect } from 'react';
import { useAuthStore } from '@/store/auth-store';

interface AuthProviderProps {
  children: React.ReactNode;
}

/**
 * Authentication provider that initializes auth state on app startup
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const initializeAuth = useAuthStore.use.initializeAuth();
  const isSystemReady = useAuthStore.use.isSystemReady();

  useEffect(() => {
    initializeAuth();
  }, [initializeAuth]);

  if (!isSystemReady) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return <>{children}</>;
}