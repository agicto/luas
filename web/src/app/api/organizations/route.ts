import {
  createOrganizationRoute,
  listOrganizationsRoute,
  privateOrganizationResponse,
} from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateOrganizationResponse(await listOrganizationsRoute(request));
}

export async function POST(request: Request) {
  return privateOrganizationResponse(await createOrganizationRoute(request));
}
