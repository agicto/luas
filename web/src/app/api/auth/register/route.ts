import { z } from 'zod';
import {
  apiInvalidInputResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { resolveAuthRoute } from '@/app/api/_shared/auth-route';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { registerWithGoApi } from '@/features/auth/server/auth-adapter-route';
import { clearApiSessionCookie } from '@/features/auth/server/api-session';
import { setSessionCookie } from '@/features/auth/server/session';

const registerSchema = z.object({
  name: z.string().trim().min(2).max(50),
  email: z.string().email().max(100),
  password: z.string().min(8).max(50),
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
    return apiInvalidInputResponse('Malformed JSON body');
  }

  const parsed = registerSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid registration payload', parsed.error);
  }

  if (resolution.backend === 'go-api') {
    return registerWithGoApi(request, parsed.data);
  }

  const { name, email } = parsed.data;
  const user = {
    id: crypto.randomUUID(),
    email,
    name,
  };

  await clearApiSessionCookie();
  await setSessionCookie(user);

  return apiSuccessResponse({
    user,
  });
}
