import 'server-only';

import { NextResponse } from 'next/server';

import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { serverEnv } from '@/config/server-env';
import { ApiErrorCode } from '@/http/codes';
import { GoApiClient, type GoApiError, type GoApiMethod } from './go-api-client';

interface PublicGoApiRequest {
  method: GoApiMethod;
  path: string;
  ifNoneMatch?: string;
  searchParams?: URLSearchParams;
}

let client: GoApiClient | undefined;

// forwardPublicGoApi forwards one fixed code-owned public operation without credentials.
export async function forwardPublicGoApi(
  request: Request,
  options: PublicGoApiRequest
): Promise<NextResponse> {
  if (!serverEnv.API_UPSTREAM_URL) {
    return publicErrorResponse({
      cause: 'unavailable',
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'API adapter is unavailable',
      requestId: crypto.randomUUID(),
    });
  }

  client ??= new GoApiClient({
    apiUrl: serverEnv.API_UPSTREAM_URL,
    timeoutMs: serverEnv.API_UPSTREAM_TIMEOUT_MS,
    maxResponseBytes: serverEnv.API_UPSTREAM_MAX_RESPONSE_BYTES,
    clientIpHeader: serverEnv.API_CLIENT_IP_HEADER,
  });
  const result = await client.request({
    ...options,
    incomingHeaders: request.headers,
    signal: request.signal,
  });
  if (!result.ok) {
    return publicErrorResponse(result.error);
  }

  const headers = new Headers(result.data.responseHeaders);
  headers.set('x-request-id', result.data.requestId);
  if (result.data.envelope === null) {
    return new NextResponse(null, { status: result.data.status, headers });
  }
  return NextResponse.json(result.data.envelope, {
    status: result.data.status,
    headers,
  });
}

function publicErrorResponse(error: GoApiError): NextResponse {
  const headers = new Headers(error.responseHeaders);
  headers.set('cache-control', 'no-store');
  headers.set('pragma', 'no-cache');
  return apiErrorResponse({
    status: error.status,
    errorCode: error.errorCode,
    message:
      error.cause === 'timeout'
        ? 'API request timed out'
        : error.cause === 'upstream'
          ? publicUpstreamMessage(error.status)
          : 'API service unavailable',
    requestId: error.requestId,
    headers,
  });
}

function publicUpstreamMessage(status: number): string {
  if (status === 404) return 'Resource not found';
  if (status === 429) return 'Too many requests';
  if (status === 400 || status === 422) return 'Invalid API input';
  return 'API service unavailable';
}
