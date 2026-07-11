import { AuthGuard } from '@/features/auth';
import { Toaster } from '@/components/ui/sonner';
import { AuthenticatedProviders } from '@/providers/authenticated-providers';

/**
 * Protected layout for authenticated routes.
 */
export default function ProtectedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AuthenticatedProviders>
      <AuthGuard>{children}</AuthGuard>
      <Toaster richColors position="top-right" />
    </AuthenticatedProviders>
  );
}
