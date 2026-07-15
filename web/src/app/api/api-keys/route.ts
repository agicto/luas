import {
  createApiKeyRoute,
  listApiKeysRoute,
  privateApiKeyResponse,
} from '@/features/api-key/server/api-key-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateApiKeyResponse(await listApiKeysRoute(request));
}

export async function POST(request: Request) {
  return privateApiKeyResponse(await createApiKeyRoute(request));
}
