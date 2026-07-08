import { z } from 'zod';
import {
  apiErrorResponse,
  apiInvalidInputResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { authConfig } from '@/config/auth';
import { setSessionCookie } from '@/features/auth/server/session';
import { ApiErrorCode } from '@/http/codes';

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
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

  const parsed = loginSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid login payload', parsed.error);
  }

  const { email, password } = parsed.data;
  const { demoUser } = authConfig;

  if (email !== demoUser.email || password !== demoUser.password) {
    return apiErrorResponse({
      status: 401,
      errorCode: ApiErrorCode.AUTH_INVALID_CREDENTIALS,
      message: 'Invalid email or password',
    });
  }

  const user = {
    id: demoUser.id,
    email: demoUser.email,
    name: demoUser.name,
    role: demoUser.role,
  };

  await setSessionCookie(user);

  return apiSuccessResponse({
    user,
  });
}
