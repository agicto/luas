import { NextResponse } from 'next/server';
import { z } from 'zod';
import { setSessionCookie } from '@/features/auth/server/session';
import { ErrorCode } from '@/http/codes';

const registerSchema = z.object({
  name: z.string().trim().min(2).max(80),
  email: z.string().email(),
  password: z.string().min(8).max(128),
});

export async function POST(request: Request) {
  const payload = await request.json().catch(() => null);
  const parsed = registerSchema.safeParse(payload);

  if (!parsed.success) {
    return NextResponse.json(
      {
        error: 'Invalid registration payload',
        code: ErrorCode.INVALID_PARAMS,
      },
      { status: 400 }
    );
  }

  const { name, email } = parsed.data;
  const user = {
    id: crypto.randomUUID(),
    email,
    name,
    role: 'member' as const,
  };

  await setSessionCookie(user);

  return NextResponse.json({
    data: {
      user,
    },
  });
}
