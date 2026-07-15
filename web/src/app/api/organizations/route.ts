import {
  createOrganizationRoute,
  listOrganizationsRoute,
} from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

export const GET = listOrganizationsRoute;
export const POST = createOrganizationRoute;
