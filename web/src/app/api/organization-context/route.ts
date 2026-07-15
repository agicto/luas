import {
  privateOrganizationResponse,
  resolveOrganizationContextRoute,
} from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateOrganizationResponse(
    await resolveOrganizationContextRoute(request)
  );
}
