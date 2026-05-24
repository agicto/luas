import { NextResponse } from 'next/server';
import { clearSessionCookie, getSessionUser } from '@/features/auth/server/session';
import { ErrorCode } from '@/http/codes';

export async function GET() {
  const user = await getSessionUser();

  if (!user) {
    await clearSessionCookie();

    return NextResponse.json(
      {
        error: 'Session expired',
        code: ErrorCode.SESSION_EXPIRED,
      },
      { status: 401 }
    );
  }

  return NextResponse.json({
    data: {
      user,
    },
  });
}
