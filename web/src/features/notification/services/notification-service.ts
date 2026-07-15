import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  notificationPageEnvelopeSchema,
  notificationPreferenceSchema,
  notificationReadStateResultSchema,
  notificationSchema,
  notificationStatusSchema,
} from '@/features/notification/schemas';
import type {
  MarkNotificationsReadInput,
  NotificationFilter,
  NotificationItem,
  NotificationPage,
  NotificationPreference,
  NotificationReadStateResult,
  NotificationStatus,
  ReplaceNotificationReadStateInput,
} from '@/features/notification/types';

interface ListOptions {
  page?: number;
  perPage?: number;
  status?: NotificationFilter;
}

export const notificationService = {
  async list({ page = 1, perPage = 10, status = 'all' }: ListOptions = {}): Promise<NotificationPage> {
    const value = await request.getEnvelope<unknown>('/notifications', {
      params: { page, per_page: perPage, status },
    });
    return parseNotificationPage(value);
  },

  async status(): Promise<NotificationStatus> {
    return parseNotificationStatus(await request.get<unknown>('/notification-status'));
  },

  async replaceReadState(
    notificationId: number,
    input: ReplaceNotificationReadStateInput
  ): Promise<NotificationItem> {
    const value = await request.patch<unknown, ReplaceNotificationReadStateInput>(
      `/notifications/${notificationId}`,
      input
    );
    return parseNotification(value);
  },

  async markReadThrough(throughId: number): Promise<NotificationReadStateResult> {
    const input: MarkNotificationsReadInput = { through_id: throughId };
    return parseNotificationReadStateResult(
      await request.put<unknown, MarkNotificationsReadInput>('/notification-read-state', input)
    );
  },

  async preference(): Promise<NotificationPreference> {
    return parseNotificationPreference(
      await request.get<unknown>('/notification-preferences')
    );
  },

  async replacePreference(input: NotificationPreference): Promise<NotificationPreference> {
    return parseNotificationPreference(
      await request.put<unknown, NotificationPreference>('/notification-preferences', input)
    );
  },
};

export function parseNotificationPage(value: unknown): NotificationPage {
  const parsed = notificationPageEnvelopeSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return {
    items: parsed.data.data,
    meta: parsed.data.meta,
    links: parsed.data.links,
  };
}

export function parseNotification(value: unknown): NotificationItem {
  const parsed = notificationSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseNotificationStatus(value: unknown): NotificationStatus {
  const parsed = notificationStatusSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseNotificationReadStateResult(value: unknown): NotificationReadStateResult {
  const parsed = notificationReadStateResultSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseNotificationPreference(value: unknown): NotificationPreference {
  const parsed = notificationPreferenceSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Notification service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}
