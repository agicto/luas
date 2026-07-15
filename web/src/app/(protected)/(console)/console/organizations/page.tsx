import { notFound } from 'next/navigation';

import { isWebFeatureEnabled } from '@/config/features';
import { OrganizationDirectory } from '@/features/organization/components/organization-directory';

export default function OrganizationsPage() {
  if (!isWebFeatureEnabled('organization')) notFound();
  return <OrganizationDirectory />;
}
