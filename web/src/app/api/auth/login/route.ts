import { NextResponse } from 'next/server';
import { z } from 'zod';
import { authConfig } from '@/config/auth';
import { setSessionCookie } from '@/features/auth/server/session';
import { ErrorCode } from '@/http/codes';

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

export async function POST(request: Request) {
  const payload = await request.json().catch(() => null);
  const parsed = loginSchema.safeParse(payload);

  if (!parsed.success) {
    return NextResponse.json(
      {
        error: 'Invalid login payload',
        code: ErrorCode.INVALID_PARAMS,
      },
      { status: 400 }
    );
  }

  const { email, password } = parsed.data;
  const { demoUser } = authConfig;

  if (email !== demoUser.email || password !== demoUser.password) {
    return NextResponse.json(
      {
        error: 'Invalid email or password',
        code: ErrorCode.INVALID_CREDENTIALS,
      },
      { status: 401 }
    );
  }

  const user = {
    id: demoUser.id,
    email: demoUser.email,
    name: demoUser.name,
    role: demoUser.role,
  };

  await setSessionCookie(user);

  return NextResponse.json({
    data: {
      user,
    },
  });
}
