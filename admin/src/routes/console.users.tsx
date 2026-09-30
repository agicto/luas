import { createFileRoute, redirect } from '@tanstack/react-router';
import { z } from 'zod';
import { isFeatureEnabled } from '@/config/env';
import { UsersPage } from '@/features/users/components/users-page';
import { userStatusFilters } from '@/features/users/types';

const searchSchema = z.object({
  q: z.string().trim().max(100).optional().catch(undefined),
  status: z.enum(userStatusFilters).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
});

export const Route = createFileRoute('/console/users')({
  validateSearch: searchSchema,
  beforeLoad: () => {
    if (!isFeatureEnabled('operator')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function UsersRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <UsersPage
        search={search}
        onSearchChange={(next) => void navigate({ search: next, replace: true })}
      />
    );
  },
});
