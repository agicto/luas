import { NextResponse } from 'next/server';
import { clearSessionCookie } from '@/features/auth/server/session';

export async function POST() {
  await clearSessionCookie();

  return NextResponse.json({
    data: {
      success: true,
    },
  });
}
