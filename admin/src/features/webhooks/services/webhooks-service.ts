import { http } from '@/http/client';
import {
  webhookAttemptPageSchema,
  webhookDeliveryPageSchema,
  webhookDeliverySchema,
  webhookEndpointPageSchema,
  type WebhookAttemptPage,
  type WebhookDelivery,
  type WebhookDeliveryFilter,
  type WebhookDeliveryPage,
  type WebhookEndpointPage,
} from '@/features/webhooks/types';

const organizationPath = (organizationId: number) => `/v1/operator/organizations/${organizationId}`;

export const webhooksService = {
  endpoints(
    organizationId: number,
    page: number,
    signal?: AbortSignal,
  ): Promise<WebhookEndpointPage> {
    const params = new URLSearchParams({ page: String(page), per_page: '20' });
    return http.get(`${organizationPath(organizationId)}/webhook-endpoints?${params.toString()}`, {
      responseMode: 'json',
      schema: webhookEndpointPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
  deliveries(
    organizationId: number,
    filter: WebhookDeliveryFilter,
    signal?: AbortSignal,
  ): Promise<WebhookDeliveryPage> {
    const params = new URLSearchParams({ page: String(filter.page), per_page: '20' });
    if (filter.status) {
      params.set('status', filter.status);
    }
    return http.get(`${organizationPath(organizationId)}/webhook-deliveries?${params.toString()}`, {
      responseMode: 'json',
      schema: webhookDeliveryPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
  attempts(
    organizationId: number,
    deliveryId: number,
    signal?: AbortSignal,
  ): Promise<WebhookAttemptPage> {
    return http.get(
      `${organizationPath(organizationId)}/webhook-deliveries/${deliveryId}/attempts?per_page=50`,
      {
        responseMode: 'json',
        schema: webhookAttemptPageSchema,
        ...(signal ? { signal } : {}),
      },
    );
  },
  replay(organizationId: number, deliveryId: number): Promise<WebhookDelivery> {
    return http.post(
      `${organizationPath(organizationId)}/webhook-deliveries/${deliveryId}/replay`,
      undefined,
      { schema: webhookDeliverySchema },
    );
  },
};
