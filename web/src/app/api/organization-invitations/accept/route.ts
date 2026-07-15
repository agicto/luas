import { acceptOrganizationInvitationRoute } from '@/features/organization/server/organization-lifecycle-route';
import { privateOrganizationResponse } from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

export async function POST(request: Request) {
  return privateOrganizationResponse(
    await acceptOrganizationInvitationRoute(request)
  );
}
