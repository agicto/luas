import {
  createWebhookEndpointRoute,
  listWebhookEndpointsRoute,
  privateWebhookResponse,
} from '@/features/webhook/server/webhook-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateWebhookResponse(await listWebhookEndpointsRoute(request));
}

export async function POST(request: Request) {
  return privateWebhookResponse(await createWebhookEndpointRoute(request));
}
