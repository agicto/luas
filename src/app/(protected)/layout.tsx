import { AuthGuard } from '@/features/auth';

/**
 * Protected layout for authenticated routes.
 */
export default function ProtectedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AuthGuard>{children}</AuthGuard>;
}
