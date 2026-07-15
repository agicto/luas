import {
  getOrganizationRoute,
  updateOrganizationRoute,
} from '@/features/organization/server/organization-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ organizationId?: string | string[] }>;
}

async function organizationId(context: RouteContext): Promise<string> {
  const value = (await context.params).organizationId;
  return Array.isArray(value) ? value[0] ?? '' : value ?? '';
}

export async function GET(request: Request, context: RouteContext) {
  return getOrganizationRoute(request, await organizationId(context));
}

export async function PATCH(request: Request, context: RouteContext) {
  return updateOrganizationRoute(request, await organizationId(context));
}
