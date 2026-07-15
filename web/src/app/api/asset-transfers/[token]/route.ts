import {
  acceptAssetTransferRoute,
  downloadAssetTransferRoute,
  privateAssetResponse,
} from '@/features/asset/server/asset-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ token: string }>;
}

export async function PUT(request: Request, context: RouteContext) {
  const { token } = await context.params;
  return privateAssetResponse(await acceptAssetTransferRoute(request, token));
}

export async function GET(_request: Request, context: RouteContext) {
  const { token } = await context.params;
  return privateAssetResponse(downloadAssetTransferRoute(token));
}
