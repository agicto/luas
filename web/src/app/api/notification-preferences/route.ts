import {
  getNotificationPreferenceRoute,
  privateNotificationResponse,
  replaceNotificationPreferenceRoute,
} from '@/features/notification/server/notification-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateNotificationResponse(await getNotificationPreferenceRoute(request));
}

export async function PUT(request: Request) {
  return privateNotificationResponse(await replaceNotificationPreferenceRoute(request));
}
