import { deleteAssetRoute, privateAssetResponse } from '@/features/asset/server/asset-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ assetId: string }>;
}

export async function DELETE(request: Request, context: RouteContext) {
  const { assetId } = await context.params;
  return privateAssetResponse(await deleteAssetRoute(request, assetId));
}
