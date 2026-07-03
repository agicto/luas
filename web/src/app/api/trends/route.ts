import { NextResponse } from 'next/server';
import { z } from 'zod';

import {
  apiErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { guardMockBffRoute } from '@/app/api/_shared/mock-bff';
import { serverEnv } from '@/config/env.server';
import { ApiErrorCode } from '@/http/codes';

const listQuerySchema = z.object({
  status: z.enum(['new', 'candidate', 'selected', 'rejected', 'archived']).optional(),
  channel: z.string().max(80).optional(),
  search: z.string().max(160).optional(),
  recommended: z.coerce.boolean().optional(),
  page: z.coerce.number().int().min(1).optional(),
  per_page: z.coerce.number().int().min(1).max(100).optional(),
});

interface BackendPaginatedResponse<T> {
  code?: number;
  message?: string;
  data?: T[];
  meta?: unknown;
  links?: unknown;
}

function trendApiURL(path: string, searchParams?: URLSearchParams): URL {
  const base = serverEnv.TREND_API_URL.endsWith('/')
    ? serverEnv.TREND_API_URL
    : `${serverEnv.TREND_API_URL}/`;
  const url = new URL(path.replace(/^\//, ''), base);

  if (searchParams) {
    searchParams.forEach((value, key) => {
      url.searchParams.set(key, value);
    });
  }

  return url;
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

export async function GET(request: Request) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const requestURL = new URL(request.url);
  const parsed = listQuerySchema.safeParse(Object.fromEntries(requestURL.searchParams.entries()));

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid trend query parameters', parsed.error);
  }

  const searchParams = new URLSearchParams();
  Object.entries(parsed.data).forEach(([key, value]) => {
    if (value !== undefined) {
      searchParams.set(key, String(value));
    }
  });

  try {
    const response = await fetch(trendApiURL('/trends', searchParams), {
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    });

    if (!response.ok) {
      return backendErrorResponse(response);
    }

    const payload = (await response.json()) as BackendPaginatedResponse<unknown>;

    return NextResponse.json({
      data: {
        items: payload.data ?? [],
        meta: payload.meta ?? null,
        links: payload.links ?? null,
      },
    });
  } catch {
    return apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Trend API is unavailable',
    });
  }
}
