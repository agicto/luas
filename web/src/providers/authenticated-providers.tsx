'use client';

import { type PropsWithChildren } from 'react';

import { AuthProvider } from './auth-provider';
import { QueryProvider } from './query-provider';

/**
 * Providers required by authenticated or data-mutating route groups.
 */
export function AuthenticatedProviders({ children }: PropsWithChildren) {
  return (
    <QueryProvider>
      <AuthProvider>{children}</AuthProvider>
    </QueryProvider>
  );
}
