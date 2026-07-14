import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { resolveAuthRoute } from '@/app/api/_shared/auth-route';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { getCurrentGoApiSession } from '@/features/auth/server/auth-adapter-route';
import { clearSessionCookie, getSessionUser } from '@/features/auth/server/session';
import { ApiErrorCode } from '@/http/codes';

export const runtime = 'nodejs';

export async function GET(request: Request) {
  const resolution = resolveAuthRoute();

  if (!resolution.available) {
    return resolution.response;
  }
  if (resolution.backend === 'go-api') {
    return getCurrentGoApiSession(request);
  }

  const user = await getSessionUser();

  if (!user) {
    await clearSessionCookie();

    return apiErrorResponse({
      status: 401,
      errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
      message: 'Session expired',
    });
  }

  return apiSuccessResponse({
    user,
  });
}
