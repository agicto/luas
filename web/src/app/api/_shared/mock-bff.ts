import type { NextResponse } from 'next/server';

import {
  isMockBffEnabled,
  type MockBffEnvironment,
} from '@/config/mock-bff';
import { ApiErrorCode } from '@/http/codes';
import { apiErrorResponse } from './error-response';

export { isMockBffEnabled } from '@/config/mock-bff';

export function guardMockBffRoute(
  environment?: MockBffEnvironment
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
