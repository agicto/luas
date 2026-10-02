import { z } from 'zod';
import { pageSchema } from '@/http/pagination';

export const notificationDeliveryStatuses = [
  'pending',
  'processing',
  'delivered',
  'failed',
] as const;
export const notificationChannels = ['in_app', 'email'] as const;
export type NotificationDeliveryStatus = (typeof notificationDeliveryStatuses)[number];
export type NotificationChannel = (typeof notificationChannels)[number];

export const notificationDeliverySchema = z.object({
  id: z.number().int().positive(),
  notification_id: z.number().int().positive(),
  user_id: z.number().int().positive(),
  kind: z.string(),
  channel: z.enum(notificationChannels),
  status: z.enum(notificationDeliveryStatuses),
  attempts: z.number().int().min(0),
  last_failure_code: z.string(),
  available_at: z.string(),
  delivered_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const notificationDeliveryPageSchema = pageSchema(notificationDeliverySchema);

export type NotificationDelivery = z.infer<typeof notificationDeliverySchema>;
export type NotificationDeliveryPage = z.infer<typeof notificationDeliveryPageSchema>;

export interface NotificationDeliverySearch {
  status?: NotificationDeliveryStatus | undefined;
  channel?: NotificationChannel | undefined;
  user_id?: number | undefined;
  page?: number | undefined;
}
