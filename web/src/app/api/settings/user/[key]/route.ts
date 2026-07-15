import {
  privateSettingResponse,
  resetUserSettingRoute,
  setUserSettingRoute,
} from '@/features/setting/server/setting-route';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: Promise<{ key: string }>;
}

export async function PATCH(request: Request, context: RouteContext) {
  const { key } = await context.params;
  return privateSettingResponse(await setUserSettingRoute(request, key));
}

export async function DELETE(request: Request, context: RouteContext) {
  const { key } = await context.params;
  return privateSettingResponse(await resetUserSettingRoute(request, key));
}
