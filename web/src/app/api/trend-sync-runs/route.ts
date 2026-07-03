import { NextResponse } from 'next/server';

import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { serverEnv } from '@/config/env.server';
import { ApiErrorCode } from '@/http/codes';

interface BackendResponse<T> {
  code?: number;
  message?: string;
  data?: T;
}

function trendApiURL(path: string): URL {
  const base = serverEnv.TREND_API_URL.endsWith('/')
    ? serverEnv.TREND_API_URL
    : `${serverEnv.TREND_API_URL}/`;
  return new URL(path.replace(/^\//, ''), base);
}

async function backendErrorResponse(response: Response) {
  const body = await response.json().catch(() => null);

  return NextResponse.json(
    body ?? {
      code: response.status,
      error_code: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Trend API request failed',
    },
    { status: response.status }
  );
}

export async function POST() {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  try {
    const response = await fetch(trendApiURL('/trend-sync-runs'), {
      method: 'POST',
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    });

    if (!response.ok) {
      return backendErrorResponse(response);
    }

    const payload = (await response.json()) as BackendResponse<unknown>;

    return NextResponse.json(
      {
        data: payload.data ?? null,
      },
      { status: 201 }
    );
  } catch {
    return apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Trend API is unavailable',
    });
  }
}
