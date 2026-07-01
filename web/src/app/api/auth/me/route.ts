import { NextResponse } from 'next/server';
import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { clearSessionCookie, getSessionUser } from '@/features/auth/server/session';
import { ApiErrorCode } from '@/http/codes';

export async function GET() {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
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

  return NextResponse.json({
    data: {
      user,
    },
  });
}
