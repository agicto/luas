import { privateOrganizationResponse } from '@/features/organization/server/organization-route';
import { listPermissionsRoute } from '@/features/permission/server/permission-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateOrganizationResponse(await listPermissionsRoute(request));
}
