
/**
 * Normal Layout - Public routes
 * 
 * All routes under (normal)/* are now publicly accessible.
 */
export default function NormalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
