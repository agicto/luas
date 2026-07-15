'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { webhookService } from '@/features/webhook/services/webhook-service';
import type { WebhookEndpoint, WebhookEndpointInput } from '@/features/webhook/types';

export const webhookKeys = {
  all: ['webhooks'] as const,
  eventTypes: (organizationId: number) =>
    [...webhookKeys.all, organizationId, 'event-types'] as const,
  endpoints: (organizationId: number) => [...webhookKeys.all, organizationId, 'endpoints'] as const,
  endpointPage: (organizationId: number, page: number) =>
    [...webhookKeys.endpoints(organizationId), page] as const,
  deliveries: (organizationId: number) =>
    [...webhookKeys.all, organizationId, 'deliveries'] as const,
  deliveryPage: (organizationId: number, page: number) =>
    [...webhookKeys.deliveries(organizationId), page] as const,
  attempts: (organizationId: number, deliveryId: number) =>
    [...webhookKeys.all, organizationId, 'deliveries', deliveryId, 'attempts'] as const,
};

export function useWebhookEventTypes(organizationId: number) {
  return useQuery({
    queryKey: webhookKeys.eventTypes(organizationId),
    queryFn: () => webhookService.eventTypes(organizationId),
    enabled: validId(organizationId),
    staleTime: Number.POSITIVE_INFINITY,
  });
}

export function useWebhookEndpoints(organizationId: number, page = 1) {
  return useQuery({
    queryKey: webhookKeys.endpointPage(organizationId, page),
    queryFn: () => webhookService.endpoints(organizationId, { page, perPage: 25 }),
    enabled: validId(organizationId) && validId(page),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
  });
}

export function useWebhookDeliveries(organizationId: number, page = 1) {
  return useQuery({
    queryKey: webhookKeys.deliveryPage(organizationId, page),
    queryFn: () => webhookService.deliveries(organizationId, { page, perPage: 25 }),
    enabled: validId(organizationId) && validId(page),
    placeholderData: keepPreviousData,
    refetchInterval: 15_000,
  });
}

export function useWebhookAttempts(organizationId: number, deliveryId: number | null) {
  return useQuery({
    queryKey: webhookKeys.attempts(organizationId, deliveryId ?? 0),
    queryFn: () => webhookService.attempts(organizationId, deliveryId ?? 0),
    enabled: validId(organizationId) && deliveryId !== null && validId(deliveryId),
    staleTime: 15_000,
  });
}

export function useCreateWebhookEndpoint(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: WebhookEndpointInput) => webhookService.create(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: webhookKeys.endpoints(organizationId) }),
  });
}

export function useUpdateWebhookEndpoint(organizationId: number) {
  return useEndpointMutation(
    organizationId,
    ({ endpoint, input }: { endpoint: WebhookEndpoint; input: WebhookEndpointInput }) =>
      webhookService.update(organizationId, endpoint.id, input, endpoint.version)
  );
}

export function useReplaceWebhookEndpointStatus(organizationId: number) {
  return useEndpointMutation(
    organizationId,
    ({ endpoint, enabled }: { endpoint: WebhookEndpoint; enabled: boolean }) =>
      webhookService.replaceStatus(organizationId, endpoint.id, enabled, endpoint.version)
  );
}

export function useDeleteWebhookEndpoint(organizationId: number) {
  return useEndpointMutation(organizationId, (endpoint: WebhookEndpoint) =>
    webhookService.remove(organizationId, endpoint.id, endpoint.version)
  );
}

export function useRotateWebhookEndpointSecret(organizationId: number) {
  return useEndpointMutation(organizationId, (endpoint: WebhookEndpoint) =>
    webhookService.rotate(organizationId, endpoint.id, endpoint.version)
  );
}

export function useTestWebhookEndpoint(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      endpoint,
      idempotencyKey,
    }: {
      endpoint: WebhookEndpoint;
      idempotencyKey: string;
    }) => webhookService.test(organizationId, endpoint.id, idempotencyKey),
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: webhookKeys.deliveries(organizationId) }),
  });
}

function useEndpointMutation<TVariables, TResult>(
  organizationId: number,
  mutationFn: (variables: TVariables) => Promise<TResult>
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: webhookKeys.endpoints(organizationId) }),
        queryClient.invalidateQueries({ queryKey: webhookKeys.deliveries(organizationId) }),
      ]),
  });
}

function validId(value: number): boolean {
  return Number.isSafeInteger(value) && value > 0;
}
