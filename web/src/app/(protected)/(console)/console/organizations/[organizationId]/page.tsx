import { notFound } from 'next/navigation';

import { isWebFeatureEnabled } from '@/config/features';
import { OrganizationOverview } from '@/features/organization/components/organization-overview';
import { organizationRouteIdSchema } from '@/features/organization/schemas';

interface OrganizationPageProps {
  params: Promise<{ organizationId: string }>;
}

export default async function OrganizationPage({ params }: OrganizationPageProps) {
  if (!isWebFeatureEnabled('organization')) notFound();
  const result = organizationRouteIdSchema.safeParse((await params).organizationId);
  if (!result.success) notFound();

  return <OrganizationOverview organizationId={result.data} />;
}
