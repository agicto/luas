import { z } from 'zod';
import { pageSchema } from '@/http/pagination';

export const webhookEndpointSchema = z.object({
  id: z.number().int().positive(),
  organization_id: z.number().int().positive(),
  name: z.string(),
  url: z.string(),
  event_types: z.array(z.string()),
  status: z.enum(['active', 'disabled']),
  disabled_reason: z.string(),
  consecutive_failures: z.number().int().min(0),
  created_at: z.string(),
  updated_at: z.string(),
});

export const webhookDeliveryStatuses = [
  'pending',
  'processing',
  'delivered',
  'failed',
  'canceled',
] as const;
export type WebhookDeliveryStatus = (typeof webhookDeliveryStatuses)[number];

export const webhookDeliverySchema = z.object({
  id: z.number().int().positive(),
  endpoint_id: z.number().int().positive(),
  message_id: z.string(),
  event_type: z.string(),
  status: z.enum(webhookDeliveryStatuses),
  attempt_count: z.number().int().min(0),
  replay_count: z.number().int().min(0),
  http_status: z.number().int().nullable(),
  failure_code: z.string(),
  available_at: z.string(),
  delivered_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const webhookAttemptSchema = z.object({
  id: z.number().int().positive(),
  delivery_id: z.number().int().positive(),
  number: z.number().int().positive(),
  outcome: z.string(),
  http_status: z.number().int().nullable(),
  failure_code: z.string(),
  duration_ms: z.number().int().min(0),
  started_at: z.string(),
  completed_at: z.string(),
});

export const webhookEndpointPageSchema = pageSchema(webhookEndpointSchema);
export const webhookDeliveryPageSchema = pageSchema(webhookDeliverySchema);
export const webhookAttemptPageSchema = pageSchema(webhookAttemptSchema);

export type WebhookEndpoint = z.infer<typeof webhookEndpointSchema>;
export type WebhookEndpointPage = z.infer<typeof webhookEndpointPageSchema>;
export type WebhookDelivery = z.infer<typeof webhookDeliverySchema>;
export type WebhookDeliveryPage = z.infer<typeof webhookDeliveryPageSchema>;
export type WebhookAttemptPage = z.infer<typeof webhookAttemptPageSchema>;

export interface WebhookDeliveryFilter {
  status?: WebhookDeliveryStatus | undefined;
  page: number;
}

/** Only a finished delivery can be replayed; the API enforces the same rule. */
export function isReplayable(delivery: WebhookDelivery): boolean {
  return (
    delivery.status === 'delivered' ||
    delivery.status === 'failed' ||
    delivery.status === 'canceled'
  );
}
