import {
  privateWebhookResponse,
  webhookEventTypesRoute,
} from '@/features/webhook/server/webhook-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateWebhookResponse(await webhookEventTypesRoute(request));
}
