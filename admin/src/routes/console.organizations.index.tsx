import { createFileRoute, redirect } from '@tanstack/react-router';
import { z } from 'zod';
import { isOperatorFeatureEnabled } from '@/config/env';
import { OrganizationsPage } from '@/features/organizations/components/organizations-page';

const searchSchema = z.object({
  q: z.string().trim().max(100).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
});

export const Route = createFileRoute('/console/organizations/')({
  validateSearch: searchSchema,
  beforeLoad: () => {
    if (!isOperatorFeatureEnabled('organization')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function OrganizationsRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <OrganizationsPage
        search={search}
        onSearchChange={(next) => void navigate({ search: next, replace: true })}
        onOpen={(organizationId) =>
          void navigate({
            to: '/console/organizations/$organizationId',
            params: { organizationId: String(organizationId) },
          })
        }
      />
    );
  },
});
