import { AuthGuard } from '@/features/auth';
import { Toaster } from '@/components/ui/sonner';
import { resolveAuthBootstrap } from '@/features/auth/server/bootstrap';
import { AuthenticatedProviders } from '@/providers/authenticated-providers';

/**
 * Protected layout for authenticated routes.
 */
export default async function ProtectedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const bootstrap = await resolveAuthBootstrap();

  return (
    <AuthenticatedProviders bootstrap={bootstrap}>
      <AuthGuard>{children}</AuthGuard>
      <Toaster richColors position="top-right" />
    </AuthenticatedProviders>
  );
}
