import { publicSettingsRoute } from '@/features/setting/server/setting-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return publicSettingsRoute(request);
}
