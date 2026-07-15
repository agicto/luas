import type { NextResponse } from 'next/server';

import {
  isMockBffEnabled,
  type MockBffEnvironment,
} from '@/config/mock-bff';
import { env } from '@/config/env';
import { serverEnv } from '@/config/server-env';
import { privateAuthResponse } from '@/features/auth/server/auth-response';
import { ApiErrorCode } from '@/http/codes';
import { apiErrorResponse } from './error-response';

export type AuthRouteBackend = 'go-api' | 'mock';

export interface AuthRouteEnvironment extends MockBffEnvironment {
  adapterEnabled: boolean;
}

type AuthRouteResolution =
  | { available: true; backend: AuthRouteBackend }
  | { available: false; response: NextResponse };

function currentEnvironment(): AuthRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    enabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

export function resolveAuthRoute(
  environment: AuthRouteEnvironment = currentEnvironment()
): AuthRouteResolution {
  if (environment.adapterEnabled) {
    return { available: true, backend: 'go-api' };
  }

  if (isMockBffEnabled(environment)) {
    return { available: true, backend: 'mock' };
  }

  return {
    available: false,
    response: privateAuthResponse(
      apiErrorResponse({
        status: 503,
        errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
        message: 'Authentication backend is unavailable',
      })
    ),
  };
}
