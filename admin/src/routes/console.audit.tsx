import { createFileRoute, redirect } from '@tanstack/react-router';
import { z } from 'zod';
import { isFeatureEnabled } from '@/config/env';
import { AuditPage } from '@/features/audit/components/audit-page';

const day = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .optional()
  .catch(undefined);

const searchSchema = z.object({
  action: z.string().trim().max(120).optional().catch(undefined),
  user_id: z.coerce.number().int().positive().optional().catch(undefined),
  from: day,
  to: day,
  cursor: z.string().max(128).optional().catch(undefined),
});

export const Route = createFileRoute('/console/audit')({
  validateSearch: searchSchema,
  beforeLoad: () => {
    if (!isFeatureEnabled('operator')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function AuditRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <AuditPage
        search={search}
        onSearchChange={(next) => void navigate({ search: next, replace: true })}
      />
    );
  },
});
