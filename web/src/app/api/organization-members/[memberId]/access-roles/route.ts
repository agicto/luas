import { privateOrganizationResponse } from '@/features/organization/server/organization-route';
import {
  getMemberAccessRolesRoute,
  replaceMemberAccessRolesRoute,
} from '@/features/permission/server/permission-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ memberId: string }>;
}

export async function GET(request: Request, context: RouteContext) {
  const { memberId } = await context.params;
  return privateOrganizationResponse(
    await getMemberAccessRolesRoute(request, memberId)
  );
}

export async function PUT(request: Request, context: RouteContext) {
  const { memberId } = await context.params;
  return privateOrganizationResponse(
    await replaceMemberAccessRolesRoute(request, memberId)
  );
}
