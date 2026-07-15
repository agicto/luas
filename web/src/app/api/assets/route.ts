import { listAssetsRoute, privateAssetResponse } from '@/features/asset/server/asset-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateAssetResponse(await listAssetsRoute(request));
}
