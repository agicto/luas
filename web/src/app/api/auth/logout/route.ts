import { NextResponse } from 'next/server';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { clearSessionCookie } from '@/features/auth/server/session';

export async function POST() {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  await clearSessionCookie();

  return NextResponse.json({
    data: {
      success: true,
    },
  });
}
