import { revokeOrganizationInvitationRoute } from '@/features/organization/server/organization-lifecycle-route';
import { privateOrganizationResponse } from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ organizationId: string; invitationId: string }>;
}

export async function DELETE(request: Request, context: RouteContext) {
  const { organizationId, invitationId } = await context.params;
  return privateOrganizationResponse(
    await revokeOrganizationInvitationRoute(
      request,
      organizationId,
      invitationId
    )
  );
}
