import {
  getNotificationStatusRoute,
  privateNotificationResponse,
} from '@/features/notification/server/notification-route';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  return privateNotificationResponse(await getNotificationStatusRoute(request));
}
