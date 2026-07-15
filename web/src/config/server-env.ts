import 'server-only';

import { env } from './env';
import { publicEnvSchema, serverEnvSchema } from './env-validation';

const parsedPublicEnv = publicEnvSchema.safeParse(env);

if (!parsedPublicEnv.success) {
  console.error('Invalid public environment variables:', parsedPublicEnv.error.format());
  throw new Error('Invalid public environment variables');
}

const parsed = serverEnvSchema.safeParse({
  API_ADAPTER_ENABLED: environmentAlias(
    'API_ADAPTER_ENABLED',
    'AUTH_ADAPTER_ENABLED'
  ),
  API_UPSTREAM_TIMEOUT_MS: environmentAlias(
    'API_UPSTREAM_TIMEOUT_MS',
    'AUTH_API_TIMEOUT_MS'
  ),
  API_UPSTREAM_MAX_RESPONSE_BYTES:
    process.env.API_UPSTREAM_MAX_RESPONSE_BYTES,
  API_UPSTREAM_URL: environmentAlias('API_UPSTREAM_URL', 'AUTH_API_URL'),
  API_CLIENT_IP_HEADER: environmentAlias(
    'API_CLIENT_IP_HEADER',
    'AUTH_CLIENT_IP_HEADER'
  ),
  MOCK_BFF_ENABLED: process.env.MOCK_BFF_ENABLED,
  SESSION_SECRET: process.env.SESSION_SECRET,
});

if (!parsed.success) {
  console.error('Invalid server environment variables:', parsed.error.format());
  throw new Error('Invalid server environment variables');
}

export const serverEnv = parsed.data;

const isProductionBuildPhase = process.env.NEXT_PHASE === 'phase-production-build';

function targetsSameOriginApiRoute(apiUrl: string, appUrl: string): boolean {
  try {
    const app = new URL(appUrl);
    const api = new URL(apiUrl, app);
    const path = api.pathname.replace(/\/+$/, '') || '/';

    return (
      api.origin === app.origin &&
      path === '/api' &&
      api.search.length === 0 &&
      api.hash.length === 0
    );
  } catch {
    return false;
  }
}

function isValidApiUpstreamUrl(value: string): boolean {
  const url = new URL(value);

  return (
    (url.protocol === 'http:' || url.protocol === 'https:') &&
    url.username.length === 0 &&
    url.password.length === 0 &&
    url.search.length === 0 &&
    url.hash.length === 0
  );
}

if (serverEnv.API_ADAPTER_ENABLED && !isProductionBuildPhase) {
  if (!serverEnv.API_UPSTREAM_URL) {
    throw new Error('API_UPSTREAM_URL must be set when API_ADAPTER_ENABLED=true');
  }

  if (!isValidApiUpstreamUrl(serverEnv.API_UPSTREAM_URL)) {
    throw new Error(
      'API_UPSTREAM_URL must be an HTTP(S) URL without credentials, query, or fragment'
    );
  }

  if (
    !targetsSameOriginApiRoute(
      env.NEXT_PUBLIC_API_URL,
      env.NEXT_PUBLIC_APP_URL
    )
  ) {
    throw new Error(
      'NEXT_PUBLIC_API_URL must target the same-origin /api route when API_ADAPTER_ENABLED=true'
    );
  }

  if (env.NODE_ENV === 'production' && !serverEnv.API_CLIENT_IP_HEADER) {
    throw new Error(
      'API_CLIENT_IP_HEADER must be set when API_ADAPTER_ENABLED=true in production runtime'
    );
  }
}

function environmentAlias(canonical: string, legacy: string): string | undefined {
  const canonicalValue = process.env[canonical];
  const legacyValue = process.env[legacy];

  if (
    canonicalValue !== undefined &&
    legacyValue !== undefined &&
    canonicalValue !== legacyValue
  ) {
    throw new Error(`${canonical} conflicts with deprecated ${legacy}`);
  }

  return canonicalValue ?? legacyValue;
}

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
