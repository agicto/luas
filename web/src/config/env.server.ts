import { z } from 'zod';

import { booleanEnv, env } from './env';

const serverEnvSchema = z.object({
  // Server-only base URL used by the local trend mock BFF proxy.
  TREND_API_URL: z.string().url().default('http://localhost:8025/v1'),

  // Development-only mock route handlers. Production runtime requires
  // explicit opt-in so the web shell does not accidentally ship demo APIs.
  MOCK_BFF_ENABLED: booleanEnv.default(false),

  // Secret used to HMAC-sign the mock session cookie. Required in
  // production; a dev-only fallback is used otherwise (with a console
  // warning) so `pnpm dev` works out of the box.
  SESSION_SECRET: z.preprocess(
    (value) => (value === '' ? undefined : value),
    z.string().min(32).optional()
  ),
});

const parsed = serverEnvSchema.safeParse({
  TREND_API_URL: process.env.TREND_API_URL,
  MOCK_BFF_ENABLED: process.env.MOCK_BFF_ENABLED,
  SESSION_SECRET: process.env.SESSION_SECRET,
});

if (!parsed.success) {
  console.error('Invalid server environment variables:', parsed.error.format());
  throw new Error('Invalid server environment variables');
}

export const serverEnv = {
  ...env,
  ...parsed.data,
};

const isProductionBuildPhase = process.env.NEXT_PHASE === 'phase-production-build';

if (serverEnv.NODE_ENV === 'production' && !isProductionBuildPhase && !serverEnv.SESSION_SECRET) {
  throw new Error('SESSION_SECRET must be set in production runtime');
}
