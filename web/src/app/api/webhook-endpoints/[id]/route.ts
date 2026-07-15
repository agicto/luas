import {
  deleteWebhookEndpointRoute,
  privateWebhookResponse,
  updateWebhookEndpointRoute,
} from '@/features/webhook/server/webhook-route';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: Promise<{ id: string }>;
}

export async function PATCH(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateWebhookResponse(await updateWebhookEndpointRoute(request, id));
}

export async function DELETE(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateWebhookResponse(await deleteWebhookEndpointRoute(request, id));
}
