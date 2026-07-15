import { z } from 'zod';
import {
  apiErrorResponse,
  apiJsonBodyErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { resolveAuthRoute } from '@/app/api/_shared/auth-route';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { loginWithGoApi } from '@/features/auth/server/auth-adapter-route';
import { clearApiSessionCookie } from '@/features/auth/server/api-session';
import { authenticateMockIdentity } from '@/features/auth/server/mock-identity';
import { setSessionCookie } from '@/features/auth/server/session';
import { ApiErrorCode } from '@/http/codes';

const loginSchema = z.object({
  email: z.string().email().max(100),
  password: z.string().min(1).max(50),
});

export const runtime = 'nodejs';

export async function POST(request: Request) {
  const resolution = resolveAuthRoute();

  if (!resolution.available) {
    return resolution.response;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  const payload = await readJsonBody(request);

  if (!payload.ok) {
    return apiJsonBodyErrorResponse(payload.error);
  }

  const parsed = loginSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid login payload', parsed.error);
  }

  const { email, password } = parsed.data;

  if (resolution.backend === 'go-api') {
    return loginWithGoApi(request, { email, password });
  }

  const user = authenticateMockIdentity(email, password);

  if (!user) {
    return apiErrorResponse({
      status: 401,
      errorCode: ApiErrorCode.AUTH_INVALID_CREDENTIALS,
      message: 'Invalid email or password',
    });
  }

  await clearApiSessionCookie();
  await setSessionCookie(user);

  return apiSuccessResponse({
    user,
  });
}
