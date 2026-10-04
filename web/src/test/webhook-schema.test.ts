import { describe, expect, it } from 'vitest';

import { webhookEventTypeListSchema } from '@/features/webhook/schemas';

describe('webhook event type catalog', () => {
  it('accepts any catalog-valid event type so a larger server catalog does not break the page', () => {
    expect(
      webhookEventTypeListSchema.safeParse(['webhook.test', 'billing.invoice_paid']).success
    ).toBe(true);
  });

  it('rejects malformed, duplicate, and empty catalogs', () => {
    for (const value of [['Webhook.Test'], ['webhooktest'], ['webhook.test', 'webhook.test'], []]) {
      expect(webhookEventTypeListSchema.safeParse(value).success).toBe(false);
    }
  });
});
