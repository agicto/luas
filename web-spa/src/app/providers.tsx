import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { queryClient } from '@/app/query-client';
import { router } from '@/app/router';
import { ThemeSync } from '@/features/preferences/components/theme-sync';

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeSync />
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
