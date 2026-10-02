import { createFileRoute, redirect } from '@tanstack/react-router';
import { z } from 'zod';
import { isOperatorFeatureEnabled } from '@/config/env';
import { NotificationDeliveriesPage } from '@/features/notification-deliveries/components/notification-deliveries-page';
import {
  notificationChannels,
  notificationDeliveryStatuses,
} from '@/features/notification-deliveries/types';

const searchSchema = z.object({
  status: z.enum(notificationDeliveryStatuses).optional().catch(undefined),
  channel: z.enum(notificationChannels).optional().catch(undefined),
  user_id: z.coerce.number().int().positive().optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
});

export const Route = createFileRoute('/console/notifications')({
  validateSearch: searchSchema,
  beforeLoad: () => {
    if (!isOperatorFeatureEnabled('notification')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function NotificationDeliveriesRoute() {
    const search = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <NotificationDeliveriesPage
        search={search}
        onSearchChange={(next) => void navigate({ search: next, replace: true })}
      />
    );
  },
});
