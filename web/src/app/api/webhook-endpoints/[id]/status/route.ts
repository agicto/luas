import {
  privateWebhookResponse,
  replaceWebhookEndpointStatusRoute,
} from '@/features/webhook/server/webhook-route';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: Promise<{ id: string }>;
}

export async function PUT(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateWebhookResponse(await replaceWebhookEndpointStatusRoute(request, id));
}
