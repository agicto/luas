import {
  createAssetUploadIntentRoute,
  privateAssetResponse,
} from '@/features/asset/server/asset-route';

export const runtime = 'nodejs';

export async function POST(request: Request) {
  return privateAssetResponse(await createAssetUploadIntentRoute(request));
}
