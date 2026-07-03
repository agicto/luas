import type { NextResponse } from 'next/server';

import { env } from '@/config/env';
import { serverEnv } from '@/config/env.server';
import { ApiErrorCode } from '@/http/codes';
import { apiErrorResponse } from './error-response';

interface MockBffEnvironment {
  nodeEnv: typeof env.NODE_ENV;
  enabled: boolean;
}

function currentMockBffEnvironment(): MockBffEnvironment {
  return {
    nodeEnv: env.NODE_ENV,
    enabled: serverEnv.MOCK_BFF_ENABLED,
  };
}

export function isMockBffEnabled(environment: MockBffEnvironment = currentMockBffEnvironment()): boolean {
  return environment.nodeEnv !== 'production' || environment.enabled;
}

export function guardMockBffRoute(
  environment: MockBffEnvironment = currentMockBffEnvironment()
): NextResponse | null {
  if (isMockBffEnabled(environment)) {
    return null;
  }

  return apiErrorResponse({
    status: 503,
    errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
    message: 'Mock BFF is disabled in production runtime',
  });
}
