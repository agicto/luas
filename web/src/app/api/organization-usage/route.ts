import {
  organizationUsageRoute,
  privateUsageResponse,
} from '@/features/usage/server/usage-route';

export async function GET(request: Request): Promise<Response> {
  return privateUsageResponse(await organizationUsageRoute(request), true);
}
