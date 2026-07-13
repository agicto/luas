'use client';

import { type PropsWithChildren } from 'react';

import type { AuthBootstrap } from '@/features/auth/types';
import { AuthProvider } from './auth-provider';
import { QueryProvider } from './query-provider';

/**
 * Providers required by authenticated or data-mutating route groups.
 */
export function AuthenticatedProviders({
  bootstrap,
  children,
}: PropsWithChildren<{ bootstrap: AuthBootstrap }>) {
  return (
    <QueryProvider>
      <AuthProvider bootstrap={bootstrap}>{children}</AuthProvider>
    </QueryProvider>
  );
}
