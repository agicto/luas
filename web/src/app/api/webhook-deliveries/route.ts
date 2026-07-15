import {
  listWebhookDeliveriesRoute,
  privateWebhookResponse,
} from '@/features/webhook/server/webhook-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateWebhookResponse(await listWebhookDeliveriesRoute(request));
}
