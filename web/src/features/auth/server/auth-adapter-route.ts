import 'server-only';

import { headers } from 'next/headers';
import type { NextResponse } from 'next/server';

import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { serverEnv } from '@/config/server-env';
import type {
  AuthBootstrap,
  LoginRequest,
  RegisterRequest,
} from '@/features/auth/types';
import {
  clearApiSessionCookie,
  getApiSessionToken,
  setApiSessionCookie,
} from '@/features/auth/server/api-session';
import {
  type AdapterError,
  GoApiAuthAdapter,
} from '@/features/auth/server/go-api-auth-adapter';
import { clearSessionCookie } from '@/features/auth/server/session';
import { ApiErrorCode } from '@/http/codes';

let adapter: GoApiAuthAdapter | undefined;

function configuredAdapter(): GoApiAuthAdapter {
  if (!serverEnv.API_UPSTREAM_URL) {
    throw new Error('API_UPSTREAM_URL is required by the production API adapter');
  }

  adapter ??= new GoApiAuthAdapter({
    apiUrl: serverEnv.API_UPSTREAM_URL,
    timeoutMs: serverEnv.API_UPSTREAM_TIMEOUT_MS,
    maxResponseBytes: serverEnv.API_UPSTREAM_MAX_RESPONSE_BYTES,
    clientIpHeader: serverEnv.API_CLIENT_IP_HEADER,
  });
  return adapter;
}

function adapterErrorResponse(error: AdapterError): NextResponse {
  return apiErrorResponse({
    status: error.status,
    errorCode: error.errorCode,
    message: error.message,
    errors: error.fieldErrors,
    headers: error.responseHeaders,
    requestId: error.requestId,
  });
}

export async function loginWithGoApi(
  request: Request,
  input: LoginRequest
): Promise<NextResponse> {
  const result = await configuredAdapter().login(
    input,
    request.headers,
    request.signal
  );

  if (!result.ok) {
    return adapterErrorResponse(result.error);
  }

  await clearSessionCookie();
  await setApiSessionCookie(
    result.data.accessToken,
    result.data.maxAgeSeconds
  );

  return apiSuccessResponse({ user: result.data.user });
}

export async function registerWithGoApi(
  request: Request,
  input: RegisterRequest
): Promise<NextResponse> {
  const result = await configuredAdapter().register(
    input,
    request.headers,
    request.signal
  );

  if (!result.ok) {
    return adapterErrorResponse(result.error);
  }

  await clearSessionCookie();
  await setApiSessionCookie(
    result.data.accessToken,
    result.data.maxAgeSeconds
  );

  return apiSuccessResponse({ user: result.data.user });
}

export async function currentGoApiSession(
  incomingHeaders: Headers,
  signal?: AbortSignal
) {
  const accessToken = await getApiSessionToken();
  if (!accessToken) {
    return {
      ok: false as const,
      error: {
        status: 401,
        errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
        message: 'Authentication required',
      } satisfies AdapterError,
    };
  }

  return configuredAdapter().profile(accessToken, incomingHeaders, signal);
}

export async function getCurrentGoApiSession(
  request: Request
): Promise<NextResponse> {
  const result = await currentGoApiSession(request.headers, request.signal);

  if (!result.ok) {
    if (result.error.status === 401) {
      await clearApiSessionCookie();
    }
    return adapterErrorResponse(result.error);
  }

  return apiSuccessResponse(result.data);
}

export async function logoutFromGoApi(): Promise<NextResponse> {
  await clearApiSessionCookie();
  await clearSessionCookie();

  return apiSuccessResponse({ success: true as const });
}

export async function resolveGoApiAuthBootstrap(): Promise<AuthBootstrap> {
  const incomingHeaders = new Headers(await headers());
  const result = await currentGoApiSession(incomingHeaders);

  if (result.ok) {
    return { status: 'authenticated', user: result.data.user };
  }
  if (
    result.error.errorCode === ApiErrorCode.AUTH_FORBIDDEN ||
    result.error.errorCode === ApiErrorCode.AUTH_ACCOUNT_DISABLED ||
    result.error.status === 403
  ) {
    return { status: 'forbidden' };
  }
  if (
    result.error.errorCode === ApiErrorCode.AUTH_UNAUTHORIZED ||
    result.error.status === 401
  ) {
    return { status: 'unauthenticated' };
  }

  return { status: 'unavailable' };
}
