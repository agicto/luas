import 'server-only';

import { env } from './env';
import { publicEnvSchema, serverEnvSchema } from './env-validation';

const parsedPublicEnv = publicEnvSchema.safeParse(env);

if (!parsedPublicEnv.success) {
  console.error('Invalid public environment variables:', parsedPublicEnv.error.format());
  throw new Error('Invalid public environment variables');
}

const parsed = serverEnvSchema.safeParse({
  MOCK_BFF_ENABLED: process.env.MOCK_BFF_ENABLED,
  SESSION_SECRET: process.env.SESSION_SECRET,
});

if (!parsed.success) {
  console.error('Invalid server environment variables:', parsed.error.format());
  throw new Error('Invalid server environment variables');
}

export const serverEnv = parsed.data;

const isProductionBuildPhase = process.env.NEXT_PHASE === 'phase-production-build';

if (
  env.NODE_ENV === 'production' &&
  !isProductionBuildPhase &&
  serverEnv.MOCK_BFF_ENABLED &&
  !serverEnv.SESSION_SECRET
) {
  throw new Error(
    'SESSION_SECRET must be set when MOCK_BFF_ENABLED=true in production runtime'
  );
}
