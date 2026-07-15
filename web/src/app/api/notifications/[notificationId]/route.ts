import {
  privateNotificationResponse,
  replaceNotificationReadStateRoute,
} from '@/features/notification/server/notification-route';

export const runtime = 'nodejs';

interface RouteContext {
  params: Promise<{ notificationId: string }>;
}

export async function PATCH(request: Request, context: RouteContext) {
  const { notificationId } = await context.params;
  return privateNotificationResponse(
    await replaceNotificationReadStateRoute(request, notificationId)
  );
}
