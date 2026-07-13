'use client';

import { useEffect, useState } from 'react';

import {
  AuthStoreContext,
  createAuthStore,
  type AuthStore,
} from '@/features/auth/store/auth-store';
import type { AuthBootstrap } from '@/features/auth/types';

interface AuthProviderProps {
  bootstrap: AuthBootstrap;
  children: React.ReactNode;
}

/**
 * Owns an isolated auth store for one protected route tree.
 * Definitive server bootstraps render immediately; client-owned sessions resolve
 * in the background and are blocked by AuthGuard until ready.
 */
export function AuthProvider({ bootstrap, children }: AuthProviderProps) {
  const [store] = useState<AuthStore>(() => createAuthStore(bootstrap));

  useEffect(() => {
    void store.getState().initializeAuth();
  }, [store]);

  return (
    <AuthStoreContext.Provider value={store}>
      {children}
    </AuthStoreContext.Provider>
  );
}
