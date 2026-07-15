import 'server-only';

import { NextResponse } from 'next/server';

import { apiErrorResponse } from '@/app/api/_shared/error-response';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import type { AuthUser } from '@/features/auth/types';
import { mockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import { parseOrganizationId } from '@/features/organization/server/organization-route';
import type { OrganizationContext } from '@/features/organization/types';
import { mockUsageStore } from '@/features/usage/server/mock-usage-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreHeaders, privateNoStoreResponse } from '@/server/http/private-response';

type UsageBackend = 'go-api' | 'mock';

interface UsageRouteEnvironment {
  adapterEnabled: boolean;
  featureEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type UsageRouteResolution =
  | { available: true; backend: UsageBackend }
  | { available: false; response: NextResponse };

type AuthenticatedUsageBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

type OrganizationUsageTarget =
  | {
      available: true;
      organizationId: number;
      authentication: Extract<AuthenticatedUsageBackend, { authenticated: true }>;
      context?: OrganizationContext;
    }
  | { available: false; response: NextResponse };

export function resolveUsageRoute(
  environment: UsageRouteEnvironment = currentEnvironment()
): UsageRouteResolution {
  if (!environment.featureEnabled) return unavailable('Usage Web feature is disabled');
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (isMockBffEnabled({ enabled: environment.mockBffEnabled, nodeEnv: environment.nodeEnv })) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Usage backend is unavailable');
}

export function privateUsageResponse<T extends Response>(response: T, organization = false): T {
  privateNoStoreResponse(response, organization ? ['Cookie', 'Organization-Id'] : ['Cookie']);
  response.headers.set('pragma', 'no-cache');
  return response;
}

export async function userUsageRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedUsageRoute();
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'usage/user',
      accessToken: route.authentication.accessToken,
    });
  }
  return apiSuccessResponse(mockUsageStore.user(route.authentication.user), {
    headers: privateUsageHeaders(false),
  });
}

export async function organizationUsageRoute(request: Request): Promise<NextResponse> {
  const target = await resolveOrganizationUsageTarget(request);
  if (!target.available) return target.response;
  if (target.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'organization-usage',
      accessToken: target.authentication.accessToken,
      organizationId: String(target.organizationId),
    });
  }
  if (target.context?.role !== 'owner' && target.context?.role !== 'admin') {
    return forbidden(privateUsageHeaders(true));
  }
  return apiSuccessResponse(mockUsageStore.organization(target.organizationId), {
    headers: privateUsageHeaders(true),
  });
}

async function resolveAuthenticatedUsageRoute(
  resolution: UsageRouteResolution = resolveUsageRoute()
) {
  if (!resolution.available) return resolution;
  const authentication = await authenticateUsageBackend(resolution.backend);
  return authentication.authenticated
    ? { available: true as const, authentication }
    : { available: false as const, response: authentication.response };
}

async function authenticateUsageBackend(backend: UsageBackend): Promise<AuthenticatedUsageBackend> {
  if (backend === 'go-api') {
    const accessToken = await getApiSessionToken();
    return accessToken
      ? { authenticated: true, backend, accessToken }
      : { authenticated: false, response: unauthenticated(privateUsageHeaders(false)) };
  }
  const user = await getSessionUser();
  return user
    ? { authenticated: true, backend, user }
    : { authenticated: false, response: unauthenticated(privateUsageHeaders(false)) };
}

async function resolveOrganizationUsageTarget(request: Request): Promise<OrganizationUsageTarget> {
  const route = await resolveAuthenticatedUsageRoute();
  if (!route.available) return route;
  const rawOrganizationId = request.headers.get('organization-id');
  if (rawOrganizationId === null) return organizationContextRequired();
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return organizationContextInvalid();
  if (route.authentication.backend === 'go-api') {
    return { available: true, organizationId, authentication: route.authentication };
  }
  const context = mockOrganizationStore.resolveContext(route.authentication.user, organizationId);
  if (!context) {
    return {
      available: false,
      response: apiErrorResponse({
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
        message: 'Organization not found',
        headers: privateUsageHeaders(true),
      }),
    };
  }
  return { available: true, organizationId, authentication: route.authentication, context };
}

function currentEnvironment(): UsageRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    featureEnabled: isWebFeatureEnabled('usage'),
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function privateUsageHeaders(organization: boolean): Headers {
  const headers = privateNoStoreHeaders(
    undefined,
    organization ? ['Cookie', 'Organization-Id'] : ['Cookie']
  );
  headers.set('pragma', 'no-cache');
  return headers;
}

function organizationContextRequired(): OrganizationUsageTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
      message: 'Organization context is required',
      headers: privateUsageHeaders(true),
    }),
  };
}

function organizationContextInvalid(): OrganizationUsageTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
      message: 'Organization context is invalid',
      headers: privateUsageHeaders(true),
    }),
  };
}

function unauthenticated(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
    headers,
  });
}

function forbidden(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 403,
    errorCode: ApiErrorCode.PERMISSION_DENIED,
    message: 'Operation forbidden',
    headers,
  });
}

function unavailable(message: string): UsageRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
      headers: { 'cache-control': 'no-store', pragma: 'no-cache' },
    }),
  };
}
