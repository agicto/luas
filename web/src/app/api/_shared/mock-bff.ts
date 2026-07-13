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

function crossOriginMutationResponse(): NextResponse {
  return apiErrorResponse({
    status: 403,
    errorCode: ApiErrorCode.AUTH_FORBIDDEN,
    message: 'Cross-origin mutation is not allowed',
  });
}

/** Reject browser writes that did not originate from this exact origin. */
export function guardSameOriginMutation(request: Request): NextResponse | null {
  const fetchSite = request.headers.get('sec-fetch-site');

  if (fetchSite && fetchSite !== 'same-origin' && fetchSite !== 'none') {
    return crossOriginMutationResponse();
  }

  const origin = request.headers.get('origin');

  if (!origin) {
    return null;
  }

  try {
    const parsedOrigin = new URL(origin);
    const requestOrigin = new URL(request.url).origin;

    if (parsedOrigin.origin !== origin || parsedOrigin.origin !== requestOrigin) {
      return crossOriginMutationResponse();
    }
  } catch {
    return crossOriginMutationResponse();
  }

  return null;
}
