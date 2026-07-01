import { NextResponse } from 'next/server';
import { z } from 'zod';
import {
  apiInvalidInputResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { setSessionCookie } from '@/features/auth/server/session';

const registerSchema = z.object({
  name: z.string().trim().min(2).max(80),
  email: z.string().email(),
  password: z.string().min(8).max(128),
});

export async function POST(request: Request) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const payload = await readJsonBody(request);

  if (!payload.ok) {
    return apiInvalidInputResponse('Malformed JSON body');
  }

  const parsed = registerSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid registration payload', parsed.error);
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
