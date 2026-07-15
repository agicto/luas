import {
  markNotificationsReadRoute,
  privateNotificationResponse,
} from '@/features/notification/server/notification-route';

export const runtime = 'nodejs';

export async function PUT(request: Request) {
  return privateNotificationResponse(await markNotificationsReadRoute(request));
}
