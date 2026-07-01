import { AuthGuard } from '@/features/auth';
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
    </AuthenticatedProviders>
  );
}
