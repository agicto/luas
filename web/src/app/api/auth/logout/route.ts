import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { clearSessionCookie } from '@/features/auth/server/session';

export async function POST() {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  await clearSessionCookie();

  return apiSuccessResponse({
    success: true,
  });
}
