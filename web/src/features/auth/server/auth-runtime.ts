import 'server-only';

import { env } from '@/config/env';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';

export type AuthRuntimeMode = 'mock-session' | 'client-session';

interface AuthRuntimeEnvironment {
  apiUrl: string;
  appUrl: string;
  mockBffEnabled: boolean;
  nodeEnv: typeof env.NODE_ENV;
}

function targetsSameOriginMockApi(apiUrl: string, appUrl: string): boolean {
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
 * Luas can inspect its signed mock cookie on the Next.js server. Real API
 * credentials belong to that API, so the browser resolves those sessions
 * unless a downstream app replaces this seam with its own server adapter.
 */
export function resolveAuthRuntimeMode(
  environment: AuthRuntimeEnvironment
): AuthRuntimeMode {
  const mockBffAvailable = isMockBffEnabled({
    nodeEnv: environment.nodeEnv,
    enabled: environment.mockBffEnabled,
  });

  return mockBffAvailable &&
    targetsSameOriginMockApi(environment.apiUrl, environment.appUrl)
    ? 'mock-session'
    : 'client-session';
}

export function getAuthRuntimeMode(): AuthRuntimeMode {
  return resolveAuthRuntimeMode({
    apiUrl: env.NEXT_PUBLIC_API_URL,
    appUrl: env.NEXT_PUBLIC_APP_URL,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  });
}
