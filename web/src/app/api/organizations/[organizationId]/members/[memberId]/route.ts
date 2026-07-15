import {
  removeOrganizationMemberRoute,
  updateOrganizationMemberRoute,
} from '@/features/organization/server/organization-lifecycle-route';
import { privateOrganizationResponse } from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ organizationId: string; memberId: string }>;
}

export async function PATCH(request: Request, context: RouteContext) {
  const { organizationId, memberId } = await context.params;
  return privateOrganizationResponse(
    await updateOrganizationMemberRoute(request, organizationId, memberId)
  );
}

export async function DELETE(request: Request, context: RouteContext) {
  const { organizationId, memberId } = await context.params;
  return privateOrganizationResponse(
    await removeOrganizationMemberRoute(request, organizationId, memberId)
  );
}
