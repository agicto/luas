import 'server-only';

import type { NextResponse } from 'next/server';

import {
  apiErrorResponse,
  apiJsonBodyErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import {
  apiNoContentResponse,
  apiPaginatedResponse,
  apiSuccessResponse,
} from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import type { AuthUser } from '@/features/auth/types';
import { apiKeyRouteIdSchema, createApiKeySchema } from '@/features/api-key/schemas';
import { mockApiKeyStore } from '@/features/api-key/server/mock-api-key-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreResponse } from '@/server/http/private-response';

type ApiKeyBackend = 'go-api' | 'mock';

interface ApiKeyRouteEnvironment {
  adapterEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type ApiKeyRouteResolution =
  { available: true; backend: ApiKeyBackend } | { available: false; response: NextResponse };

type AuthenticatedApiKeyBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

const maxCreateBodyBytes = 8 * 1_024;

export function resolveApiKeyRoute(
  environment: ApiKeyRouteEnvironment = currentEnvironment()
): ApiKeyRouteResolution {
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (
    isMockBffEnabled({
      enabled: environment.mockBffEnabled,
      nodeEnv: environment.nodeEnv,
    })
  ) {
    return { available: true, backend: 'mock' };
  }
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'API key backend is unavailable',
    }),
  };
}

export function privateApiKeyResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie']);
}

export async function listApiKeysRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveApiKeyRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateApiKeyBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const { page, perPage, searchParams } = paginationFromRequest(request);

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'api-keys',
      accessToken: authentication.accessToken,
      searchParams,
    });
  }

  const result = mockApiKeyStore.list(authentication.user, page, perPage);
  return apiPaginatedResponse(result.items, result.meta, result.links);
}

export async function createApiKeyRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveApiKeyRoute();
  if (!resolution.available) return resolution.response;
  const originGuard = guardSameOriginMutation(request);
  if (originGuard) return originGuard;
  const authentication = await authenticateApiKeyBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;

  const payload = await readJsonBody(request, maxCreateBodyBytes);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = createApiKeySchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid API key payload', parsed.error);
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'api-keys',
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { name: 'name', scopes: 'scopes', expires_at: 'expires_at' },
    });
  }

  return apiSuccessResponse(mockApiKeyStore.create(authentication.user, parsed.data), {
    status: 201,
    message: 'created',
  });
}

export async function revokeApiKeyRoute(
  request: Request,
  rawApiKeyId: string
): Promise<NextResponse> {
  const resolution = resolveApiKeyRoute();
  if (!resolution.available) return resolution.response;
  const originGuard = guardSameOriginMutation(request);
  if (originGuard) return originGuard;
  const authentication = await authenticateApiKeyBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const parsedId = apiKeyRouteIdSchema.safeParse(rawApiKeyId);
  if (!parsedId.success) return apiKeyNotFound();

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `api-keys/${parsedId.data}`,
      accessToken: authentication.accessToken,
    });
  }

  return mockApiKeyStore.revoke(authentication.user, parsedId.data)
    ? apiNoContentResponse()
    : apiKeyNotFound();
}

async function authenticateApiKeyBackend(
  backend: ApiKeyBackend
): Promise<AuthenticatedApiKeyBackend> {
  if (backend === 'go-api') {
    const accessToken = await getApiSessionToken();
    return accessToken
      ? { authenticated: true, backend, accessToken }
      : { authenticated: false, response: unauthenticated() };
  }

  const user = await getSessionUser();
  return user
    ? { authenticated: true, backend, user }
    : { authenticated: false, response: unauthenticated() };
}

function currentEnvironment(): ApiKeyRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function paginationFromRequest(request: Request) {
  const input = new URL(request.url).searchParams;
  const page = positiveInteger(input.get('page'), 1);
  const perPage = Math.min(positiveInteger(input.get('per_page'), 15), 100);
  return {
    page,
    perPage,
    searchParams: new URLSearchParams({ page: String(page), per_page: String(perPage) }),
  };
}

function positiveInteger(value: string | null, fallback: number): number {
  if (!value || !/^[1-9]\d*$/.test(value)) return fallback;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : fallback;
}

function unauthenticated(): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
  });
}

function apiKeyNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.API_KEY_NOT_FOUND,
    message: 'API key not found',
  });
}
