import {
  guardMockBffRoute,
  guardSameOriginMutation,
} from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { clearSessionCookie } from '@/features/auth/server/session';

export async function POST(request: Request) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  await clearSessionCookie();

  return apiSuccessResponse({
    success: true,
  });
}
