import { describe, expect, it } from 'vitest';

import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import {
  parseNotification,
  parseNotificationPage,
  parseNotificationPreference,
  parseNotificationStatus,
} from '@/features/notification/services/notification-service';

describe('notification browser contract', () => {
  it('parses the strict paginated notification envelope', () => {
    const page = parseNotificationPage(pageEnvelope([notification()]));
    expect(page.items[0]).toMatchObject({
      id: 42,
      kind: 'billing.invoice_paid',
      is_read: false,
    });
    expect(page.meta.total).toBe(1);
  });

  it('rejects unsafe action URLs and unknown successful fields', () => {
    expectInvalidResponse(() =>
      parseNotification({ ...notification(), action_url: '//evil.example/path' })
    );
    expectInvalidResponse(() =>
      parseNotification({ ...notification(), action_url: '/%2F%2Fevil.example/path' })
    );
    expectInvalidResponse(() =>
      parseNotification({ ...notification(), action_url: '/console/%5Csettings' })
    );
    expectInvalidResponse(() =>
      parseNotification({ ...notification(), action_url: '/console/%0Asettings' })
    );
    expectInvalidResponse(() =>
      parseNotification({ ...notification(), provider_response: 'secret' })
    );
  });

  it('rejects malformed status and preference payloads', () => {
    expectInvalidResponse(() => parseNotificationStatus({ unread_count: -1 }));
    expectInvalidResponse(() =>
      parseNotificationPreference({ in_app_enabled: true, email_enabled: 'yes' })
    );
  });

  it('keeps titles and bodies as plain data rather than HTML contracts', () => {
    const item = parseNotification({
      ...notification(),
      title: '<b>Invoice paid</b>',
      body: '<script>alert(1)</script>',
    });
    expect(item.title).toBe('<b>Invoice paid</b>');
    expect(item.body).toBe('<script>alert(1)</script>');
  });
});

function expectInvalidResponse(call: () => unknown): void {
  try {
    call();
    throw new Error('expected parser failure');
  } catch (error) {
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).errorCode).toBe(ClientErrorCode.INVALID_RESPONSE);
  }
}

function notification() {
  return {
    id: 42,
    kind: 'billing.invoice_paid',
    title: 'Invoice paid',
    body: 'Invoice 1042 was paid.',
    action_url: '/console/invoices/1042',
    is_read: false,
    read_at: null,
    created_at: '2026-07-15T14:00:00Z',
  };
}

function pageEnvelope(data: unknown[]) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 10,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}
