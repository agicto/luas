import {
  privateSettingResponse,
  resetOrganizationSettingRoute,
  setOrganizationSettingRoute,
} from '@/features/setting/server/setting-route';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: Promise<{ key: string }>;
}

export async function PATCH(request: Request, context: RouteContext) {
  const { key } = await context.params;
  return privateSettingResponse(await setOrganizationSettingRoute(request, key), true);
}

export async function DELETE(request: Request, context: RouteContext) {
  const { key } = await context.params;
  return privateSettingResponse(await resetOrganizationSettingRoute(request, key), true);
}
