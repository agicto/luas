import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { notificationDeliveriesService } from '@/features/notification-deliveries/services/notification-deliveries-service';
import type { NotificationDeliverySearch } from '@/features/notification-deliveries/types';

export function useNotificationDeliveries(search: NotificationDeliverySearch) {
  return useQuery({
    queryKey: ['operator', 'notification-deliveries', search],
    queryFn: ({ signal }) => notificationDeliveriesService.list(search, signal),
    placeholderData: keepPreviousData,
  });
}
