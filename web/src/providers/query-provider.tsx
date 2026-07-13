'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { PropsWithChildren, useState } from 'react';
import { isDev } from '@/config/env';
import { createQueryClient } from '@/config/query-client';

/**
 * React Query provider.
 * Provides an isolated QueryClient for the nearest route group.
 */
export function QueryProvider({ children }: PropsWithChildren) {
  const [client] = useState(createQueryClient);

  return (
    <QueryClientProvider client={client}>
      {children}
      {isDev ? <ReactQueryDevtools initialIsOpen={false} position="bottom" /> : null}
    </QueryClientProvider>
  );
}
