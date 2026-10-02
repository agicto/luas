import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { webhooksService } from '@/features/webhooks/services/webhooks-service';
import type { WebhookDeliveryFilter } from '@/features/webhooks/types';

const webhooksQueryKey = (organizationId: number) =>
  ['operator', 'organizations', organizationId, 'webhooks'] as const;

export function useWebhookEndpoints(organizationId: number, page: number) {
  return useQuery({
    queryKey: [...webhooksQueryKey(organizationId), 'endpoints', page],
    queryFn: ({ signal }) => webhooksService.endpoints(organizationId, page, signal),
    placeholderData: keepPreviousData,
  });
}

export function useWebhookDeliveries(organizationId: number, filter: WebhookDeliveryFilter) {
  return useQuery({
    queryKey: [...webhooksQueryKey(organizationId), 'deliveries', filter],
    queryFn: ({ signal }) => webhooksService.deliveries(organizationId, filter, signal),
    placeholderData: keepPreviousData,
  });
}

export function useWebhookAttempts(organizationId: number, deliveryId: number | null) {
  return useQuery({
    queryKey: [...webhooksQueryKey(organizationId), 'attempts', deliveryId],
    queryFn: ({ signal }) => webhooksService.attempts(organizationId, deliveryId ?? 0, signal),
    enabled: deliveryId !== null,
  });
}

export function useReplayWebhookDelivery(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deliveryId: number) => webhooksService.replay(organizationId, deliveryId),
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: [...webhooksQueryKey(organizationId), 'deliveries'],
      }),
  });
}
