import { privateApiKeyResponse, revokeApiKeyRoute } from '@/features/api-key/server/api-key-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ apiKeyId: string }>;
}

export async function DELETE(request: Request, context: RouteContext) {
  const { apiKeyId } = await context.params;
  return privateApiKeyResponse(await revokeApiKeyRoute(request, apiKeyId));
}
