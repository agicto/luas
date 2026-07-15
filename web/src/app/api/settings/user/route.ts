import { privateSettingResponse, userSettingsRoute } from '@/features/setting/server/setting-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateSettingResponse(await userSettingsRoute(request));
}
