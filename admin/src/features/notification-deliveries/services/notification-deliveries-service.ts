import { http } from '@/http/client';
import {
  notificationDeliveryPageSchema,
  type NotificationDeliveryPage,
  type NotificationDeliverySearch,
} from '@/features/notification-deliveries/types';

export const notificationDeliveriesService = {
  list(
    search: NotificationDeliverySearch,
    signal?: AbortSignal,
  ): Promise<NotificationDeliveryPage> {
    const params = new URLSearchParams({ page: String(search.page ?? 1), per_page: '50' });
    if (search.status) {
      params.set('status', search.status);
    }
    if (search.channel) {
      params.set('channel', search.channel);
    }
    if (search.user_id) {
      params.set('user_id', String(search.user_id));
    }
    return http.get(`/v1/operator/notification-deliveries?${params.toString()}`, {
      responseMode: 'json',
      schema: notificationDeliveryPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
};
