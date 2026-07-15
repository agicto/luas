import 'server-only';

import { NextResponse } from 'next/server';

import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { serverEnv } from '@/config/server-env';
import {
  clearApiSessionCookie,
  getApiSessionToken,
} from '@/features/auth/server/api-session';
import { ApiErrorCode } from '@/http/codes';
import { privateNoStoreHeaders } from '@/server/http/private-response';
import {
  GoApiClient,
  type GoApiError,
  type GoApiMethod,
} from './go-api-client';

interface AuthenticatedGoApiRequest {
  method: GoApiMethod;
  path: string;
  accessToken?: string;
  body?: unknown;
  organizationId?: string;
  searchParams?: URLSearchParams;
  fieldMap?: Readonly<Record<string, string>>;
}

let client: GoApiClient | undefined;

export async function forwardAuthenticatedGoApi(
  request: Request,
  options: AuthenticatedGoApiRequest
): Promise<NextResponse> {
  const accessToken = options.accessToken ?? await getApiSessionToken();
  if (!accessToken) {
    return apiErrorResponse({
      status: 401,
      errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
      message: 'Authentication required',
      headers: authenticatedResponseHeaders(),
    });
  }
  if (!serverEnv.API_UPSTREAM_URL) {
    return apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'API adapter is unavailable',
      headers: authenticatedResponseHeaders(),
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
    accessToken,
    incomingHeaders: request.headers,
    signal: request.signal,
  });

  if (!result.ok) {
    if (result.error.status === 401) {
      await clearApiSessionCookie();
    }
    return goApiErrorResponse(result.error);
  }

  const headers = authenticatedResponseHeaders(
    result.data.responseHeaders,
    result.data.requestId
  );
  if (result.data.envelope === null) {
    return new NextResponse(null, { status: result.data.status, headers });
  }
  return NextResponse.json(result.data.envelope, {
    status: result.data.status,
    headers,
  });
}

function goApiErrorResponse(error: GoApiError): NextResponse {
  const message = error.cause === 'timeout'
    ? 'API request timed out'
    : error.cause === 'unavailable'
      ? 'API service unavailable'
      : genericUpstreamMessage(error.status);

  return apiErrorResponse({
    status: error.status,
    errorCode: error.errorCode,
    message,
    errors: error.fieldErrors,
    requestId: error.requestId,
    headers: authenticatedResponseHeaders(
      error.responseHeaders,
      error.requestId
    ),
  });
}

function genericUpstreamMessage(status: number): string {
  if (status === 401) return 'Authentication required';
  if (status === 403) return 'Operation forbidden';
  if (status === 404) return 'Resource not found';
  if (status === 409) return 'Resource state conflict';
  if (status === 400 || status === 422) return 'Invalid API input';
  if (status === 429) return 'Too many requests';
  return 'API service unavailable';
}

function authenticatedResponseHeaders(
  upstream: Record<string, string> | undefined = undefined,
  requestId?: string
): Headers {
  const headers = privateNoStoreHeaders(upstream, ['Cookie']);
  if (requestId) {
    headers.set('x-request-id', requestId);
  }
  return headers;
}
