import { transferOrganizationOwnershipRoute } from '@/features/organization/server/organization-lifecycle-route';
import { privateOrganizationResponse } from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ organizationId: string }>;
}

export async function POST(request: Request, context: RouteContext) {
  const { organizationId } = await context.params;
  return privateOrganizationResponse(
    await transferOrganizationOwnershipRoute(request, organizationId)
  );
}
