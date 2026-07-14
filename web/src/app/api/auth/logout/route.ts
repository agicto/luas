import { resolveAuthRoute } from '@/app/api/_shared/auth-route';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { logoutFromGoApi } from '@/features/auth/server/auth-adapter-route';
import { clearApiSessionCookie } from '@/features/auth/server/api-session';
import { clearSessionCookie } from '@/features/auth/server/session';

export const runtime = 'nodejs';

export async function POST(request: Request) {
  const resolution = resolveAuthRoute();

  if (!resolution.available) {
    return resolution.response;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  if (resolution.backend === 'go-api') {
    return logoutFromGoApi();
  }

  await clearApiSessionCookie();
  await clearSessionCookie();

  return apiSuccessResponse({
    success: true,
  });
}
