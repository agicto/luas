import 'server-only';

import { env } from '@/config/env';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';

export type AuthRuntimeMode =
  | 'api-session'
  | 'mock-session'
  | 'client-session';

interface AuthRuntimeEnvironment {
  apiUrl: string;
  appUrl: string;
  apiAdapterEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: typeof env.NODE_ENV;
}

function targetsSameOriginApiRoute(apiUrl: string, appUrl: string): boolean {
  try {
    const app = new URL(appUrl);
    const api = new URL(apiUrl, app);
    const apiPath = api.pathname.replace(/\/+$/, '') || '/';

    return (
      api.origin === app.origin &&
      apiPath === '/api' &&
      api.search.length === 0 &&
      api.hash.length === 0
    );
  } catch {
    return false;
  }
}

/**
 * Selects who can authoritatively resolve the current session.
 *
 * Luas can inspect its signed mock cookie or resolve the Go API cookie through
 * the fixed production adapter. Other identity providers remain client-owned
 * unless a downstream app replaces this seam with its own server resolver.
 */
export function resolveAuthRuntimeMode(
  environment: AuthRuntimeEnvironment
): AuthRuntimeMode {
  if (
    environment.apiAdapterEnabled &&
    targetsSameOriginApiRoute(environment.apiUrl, environment.appUrl)
  ) {
    return 'api-session';
  }

  const mockBffAvailable = isMockBffEnabled({
    nodeEnv: environment.nodeEnv,
    enabled: environment.mockBffEnabled,
  });

  return mockBffAvailable &&
    targetsSameOriginApiRoute(environment.apiUrl, environment.appUrl)
    ? 'mock-session'
    : 'client-session';
}

export function getAuthRuntimeMode(): AuthRuntimeMode {
  return resolveAuthRuntimeMode({
    apiUrl: env.NEXT_PUBLIC_API_URL,
    appUrl: env.NEXT_PUBLIC_APP_URL,
    apiAdapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  });
}
