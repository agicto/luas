import 'server-only';

import type { NextResponse } from 'next/server';

import {
  apiErrorResponse,
  apiJsonBodyErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import { apiPaginatedResponse, apiSuccessResponse } from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import {
  assetRouteIdSchema,
  assetTransferTokenSchema,
  createUploadIntentSchema,
} from '@/features/asset/schemas';
import { MockAssetError, mockAssetStore } from '@/features/asset/server/mock-asset-store';
import type { AssetFilter } from '@/features/asset/types';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import type { AuthUser } from '@/features/auth/types';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreResponse } from '@/server/http/private-response';

type AssetBackend = 'go-api' | 'mock';

interface AssetRouteEnvironment {
  adapterEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type AssetRouteResolution =
  { available: true; backend: AssetBackend } | { available: false; response: NextResponse };

type AuthenticatedAssetBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

const maxIntentBodyBytes = 8 * 1_024;

export function resolveAssetRoute(
  environment: AssetRouteEnvironment = currentEnvironment()
): AssetRouteResolution {
  if (!isWebFeatureEnabled('asset')) return unavailable('Asset Web feature is disabled');
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (isMockBffEnabled({ enabled: environment.mockBffEnabled, nodeEnv: environment.nodeEnv })) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Asset backend is unavailable');
}

export function privateAssetResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie']);
}

export async function listAssetsRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute();
  if (!route.available) return route.response;
  const input = new URL(request.url).searchParams;
  const rawStatus = input.get('status') ?? 'all';
  if (!['all', 'pending', 'ready', 'rejected'].includes(rawStatus)) {
    return apiValidationErrorResponse('Invalid asset filter');
  }
  const page = positiveInteger(input.get('page'), 1);
  const perPage = Math.min(positiveInteger(input.get('per_page'), 25), 100);
  const status = rawStatus as AssetFilter;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'assets',
      accessToken: route.authentication.accessToken,
      searchParams: new URLSearchParams({ page: String(page), per_page: String(perPage), status }),
    });
  }
  const result = mockAssetStore.list(route.authentication.user, page, perPage, status);
  return apiPaginatedResponse(result.items, result.meta, result.links);
}

export async function createAssetUploadIntentRoute(request: Request): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const payload = await readJsonBody(request, maxIntentBodyBytes);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = createUploadIntentSchema.safeParse(payload.data);
  if (!parsed.success)
    return apiValidationErrorResponse('Invalid asset upload intent', parsed.error);
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'assets/upload-intents',
      accessToken: route.authentication.accessToken,
      body: parsed.data,
      fieldMap: {
        idempotency_key: 'idempotency_key',
        original_name: 'original_name',
        media_type: 'media_type',
        size_bytes: 'size_bytes',
      },
    });
  }
  try {
    return apiSuccessResponse(
      mockAssetStore.createUploadIntent(
        route.authentication.user,
        parsed.data,
        new URL(request.url).origin
      ),
      { status: 201, message: 'created' }
    );
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

export async function completeAssetRoute(
  request: Request,
  rawAssetId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const assetId = parseAssetId(rawAssetId);
  if (!assetId) return assetNotFound();
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: `assets/${assetId}/complete`,
      accessToken: route.authentication.accessToken,
    });
  }
  try {
    return apiSuccessResponse(mockAssetStore.complete(route.authentication.user, assetId));
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

export async function createAssetDownloadGrantRoute(
  request: Request,
  rawAssetId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const assetId = parseAssetId(rawAssetId);
  if (!assetId) return assetNotFound();
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: `assets/${assetId}/download-grant`,
      accessToken: route.authentication.accessToken,
    });
  }
  try {
    return apiSuccessResponse(
      mockAssetStore.downloadGrant(route.authentication.user, assetId, new URL(request.url).origin)
    );
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

export async function deleteAssetRoute(
  request: Request,
  rawAssetId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const assetId = parseAssetId(rawAssetId);
  if (!assetId) return assetNotFound();
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `assets/${assetId}`,
      accessToken: route.authentication.accessToken,
    });
  }
  try {
    mockAssetStore.delete(route.authentication.user, assetId);
    return apiSuccessResponse({ deleted: true as const });
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

export async function acceptAssetTransferRoute(
  request: Request,
  rawToken: string
): Promise<Response> {
  const resolution = resolveAssetRoute();
  if (!resolution.available) return resolution.response;
  if (resolution.backend !== 'mock') return assetNotFound();
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const token = parseTransferToken(rawToken);
  if (!token) return assetNotFound();
  try {
    await mockAssetStore.acceptUpload(token, request);
    return new Response(null, { status: 204, headers: { 'cache-control': 'private, no-store' } });
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

export function downloadAssetTransferRoute(rawToken: string): Response {
  const resolution = resolveAssetRoute();
  if (!resolution.available) return resolution.response;
  if (resolution.backend !== 'mock') return assetNotFound();
  const token = parseTransferToken(rawToken);
  if (!token) return assetNotFound();
  try {
    return mockAssetStore.download(token);
  } catch (error) {
    return mockAssetErrorResponse(error);
  }
}

async function resolveMutationRoute(request: Request) {
  const resolution = resolveAssetRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  return guard
    ? { available: false as const, response: guard }
    : resolveAuthenticatedRoute(resolution);
}

async function resolveAuthenticatedRoute(resolution: AssetRouteResolution = resolveAssetRoute()) {
  if (!resolution.available) return resolution;
  const authentication = await authenticateAssetBackend(resolution.backend);
  return authentication.authenticated
    ? { available: true as const, authentication }
    : { available: false as const, response: authentication.response };
}

async function authenticateAssetBackend(backend: AssetBackend): Promise<AuthenticatedAssetBackend> {
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

function parseAssetId(value: string): string | null {
  const parsed = assetRouteIdSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

function parseTransferToken(value: string): string | null {
  const parsed = assetTransferTokenSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

function positiveInteger(value: string | null, fallback: number): number {
  if (!value || !/^[1-9]\d*$/.test(value)) return fallback;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : fallback;
}

function currentEnvironment(): AssetRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function unauthenticated(): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
  });
}

function assetNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.ASSET_NOT_FOUND,
    message: 'Asset not found',
  });
}

function unavailable(message: string): AssetRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
    }),
  };
}

function mockAssetErrorResponse(error: unknown): NextResponse {
  if (error instanceof MockAssetError) {
    return apiErrorResponse({
      status: error.status,
      errorCode: error.errorCode,
      message: error.message,
    });
  }
  return apiErrorResponse({
    status: 503,
    errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
    message: 'Asset backend is unavailable',
  });
}
