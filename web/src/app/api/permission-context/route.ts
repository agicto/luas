import { privateOrganizationResponse } from '@/features/organization/server/organization-route';
import { getPermissionContextRoute } from '@/features/permission/server/permission-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateOrganizationResponse(await getPermissionContextRoute(request));
}
