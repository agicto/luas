import { privateOrganizationResponse } from '@/features/organization/server/organization-route';
import {
  deleteAccessRoleRoute,
  getAccessRoleRoute,
  updateAccessRoleRoute,
} from '@/features/permission/server/permission-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ roleId: string }>;
}

export async function GET(request: Request, context: RouteContext) {
  const { roleId } = await context.params;
  return privateOrganizationResponse(await getAccessRoleRoute(request, roleId));
}

export async function PATCH(request: Request, context: RouteContext) {
  const { roleId } = await context.params;
  return privateOrganizationResponse(await updateAccessRoleRoute(request, roleId));
}

export async function DELETE(request: Request, context: RouteContext) {
  const { roleId } = await context.params;
  return privateOrganizationResponse(await deleteAccessRoleRoute(request, roleId));
}
