import type { infer as Infer } from 'zod/mini';

import type {
  webhookAttemptPageEnvelopeSchema,
  webhookAttemptSchema,
  webhookDeliveryPageEnvelopeSchema,
  webhookDeliverySchema,
  webhookEndpointInputSchema,
  webhookEndpointPageEnvelopeSchema,
  webhookEndpointSchema,
  webhookEndpointSecretSchema,
  webhookEndpointStatusInputSchema,
} from './schemas';

export type WebhookEndpoint = Infer<typeof webhookEndpointSchema>;
export type WebhookEndpointSecret = Infer<typeof webhookEndpointSecretSchema>;
export type WebhookEndpointInput = Infer<typeof webhookEndpointInputSchema>;
export type WebhookEndpointStatusInput = Infer<typeof webhookEndpointStatusInputSchema>;
export type WebhookDelivery = Infer<typeof webhookDeliverySchema>;
export type WebhookAttempt = Infer<typeof webhookAttemptSchema>;

export interface WebhookPage<T> {
  items: T[];
  meta: Infer<typeof webhookEndpointPageEnvelopeSchema>['meta'];
  links: Infer<typeof webhookEndpointPageEnvelopeSchema>['links'];
}

export type WebhookEndpointPage = WebhookPage<WebhookEndpoint>;
export type WebhookDeliveryPage = WebhookPage<WebhookDelivery> & {
  meta: Infer<typeof webhookDeliveryPageEnvelopeSchema>['meta'];
  links: Infer<typeof webhookDeliveryPageEnvelopeSchema>['links'];
};
export type WebhookAttemptPage = WebhookPage<WebhookAttempt> & {
  meta: Infer<typeof webhookAttemptPageEnvelopeSchema>['meta'];
  links: Infer<typeof webhookAttemptPageEnvelopeSchema>['links'];
};
