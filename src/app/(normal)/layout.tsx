import { AuthGuard } from '@/components/auth-guard';

/**
 * Normal Layout - Protected routes requiring authentication
 * 
 * All routes under (normal)/* will require user to be logged in.
 * Unauthenticated users are redirected to /login.
 */
export default function NormalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AuthGuard>{children}</AuthGuard>;
}
