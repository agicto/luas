import { privateOrganizationResponse } from '@/features/organization/server/organization-route';
import {
  createAccessRoleRoute,
  listAccessRolesRoute,
} from '@/features/permission/server/permission-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateOrganizationResponse(await listAccessRolesRoute(request));
}

export async function POST(request: Request) {
  return privateOrganizationResponse(await createAccessRoleRoute(request));
}
