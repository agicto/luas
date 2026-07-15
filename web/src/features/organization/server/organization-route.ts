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
  apiPaginatedResponse,
  apiSuccessResponse,
} from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import {
  createOrganizationSchema,
  organizationRouteIdSchema,
  updateOrganizationSchema,
} from '@/features/organization/schemas';
import { mockOrganizationStore } from './mock-organization-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import {
  privateNoStoreHeaders,
  privateNoStoreResponse,
} from '@/server/http/private-response';

type OrganizationBackend = 'go-api' | 'mock';

type AuthenticatedOrganizationBackend =
  | { backend: 'go-api'; accessToken: string }
  | { backend: 'mock' };

interface OrganizationRouteEnvironment {
  adapterEnabled: boolean;
  featureEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type OrganizationRouteResolution =
  | { available: true; backend: OrganizationBackend }
  | { available: false; response: NextResponse };

export function resolveOrganizationRoute(
  environment: OrganizationRouteEnvironment = currentEnvironment()
): OrganizationRouteResolution {
  if (!environment.featureEnabled) {
    return unavailable('Organization Web feature is disabled');
  }
  if (environment.adapterEnabled) {
    return { available: true, backend: 'go-api' };
  }
  if (
    isMockBffEnabled({
      enabled: environment.mockBffEnabled,
      nodeEnv: environment.nodeEnv,
    })
  ) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Organization backend is unavailable');
}

export function privateOrganizationResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie']);
}

export async function listOrganizationsRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;

  const { page, perPage, searchParams } = paginationFromRequest(request);
  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'organizations',
      accessToken: authentication.accessToken,
      searchParams,
    });
  }

  const result = mockOrganizationStore.list(page, perPage);
  return apiPaginatedResponse(result.items, result.meta, result.links, noStoreHeaders());
}

export async function createOrganizationRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = createOrganizationSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid organization payload', parsed.error);
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'organizations',
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { name: 'name', slug: 'slug' },
    });
  }
  const organization = mockOrganizationStore.create(parsed.data);
  if (organization === 'slug_conflict') {
    return apiErrorResponse({
      status: 409,
      errorCode: ApiErrorCode.ORGANIZATION_SLUG_ALREADY_EXISTS,
      message: 'Organization slug already exists',
      headers: noStoreHeaders(),
    });
  }
  return apiSuccessResponse(organization, {
    status: 201,
    message: 'created',
    headers: noStoreHeaders(),
  });
}

export async function getOrganizationRoute(
  request: Request,
  rawOrganizationId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return organizationNotFound();

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `organizations/${organizationId}`,
      accessToken: authentication.accessToken,
    });
  }

  const organization = mockOrganizationStore.get(organizationId);
  return organization
    ? apiSuccessResponse(organization, { headers: noStoreHeaders() })
    : organizationNotFound();
}

export async function updateOrganizationRoute(
  request: Request,
  rawOrganizationId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return organizationNotFound();

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = updateOrganizationSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid organization update payload', parsed.error);
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `organizations/${organizationId}`,
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { name: 'name' },
    });
  }
  const organization = mockOrganizationStore.update(organizationId, parsed.data);
  return organization
    ? apiSuccessResponse(organization, { headers: noStoreHeaders() })
    : organizationNotFound();
}

export async function resolveOrganizationContextRoute(
  request: Request
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationIdHeader = request.headers.get('organization-id');
  if (organizationIdHeader === null) {
    return apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
      message: 'Organization context is required',
      headers: contextHeaders(),
    });
  }
  const organizationId = parseOrganizationId(organizationIdHeader);
  if (!organizationId) {
    return apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
      message: 'Organization context is invalid',
      headers: contextHeaders(),
    });
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'organization-context',
      accessToken: authentication.accessToken,
      organizationId: String(organizationId),
    });
  }

  const context = mockOrganizationStore.resolveContext(organizationId);
  return context
    ? apiSuccessResponse(context, { headers: contextHeaders() })
    : organizationNotFound(contextHeaders());
}

function currentEnvironment(): OrganizationRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    featureEnabled: isWebFeatureEnabled('organization'),
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

async function authenticateOrganizationBackend(
  backend: OrganizationBackend
): Promise<
  | ({ authenticated: true } & AuthenticatedOrganizationBackend)
  | { authenticated: false; response: NextResponse }
> {
  if (backend === 'go-api') {
    const accessToken = await getApiSessionToken();
    return accessToken
      ? { authenticated: true, backend, accessToken }
      : { authenticated: false, response: unauthenticated() };
  }

  return await getSessionUser()
    ? { authenticated: true, backend }
    : { authenticated: false, response: unauthenticated() };
}

function paginationFromRequest(request: Request) {
  const input = new URL(request.url).searchParams;
  const page = positiveInteger(input.get('page'), 1);
  const perPage = Math.min(positiveInteger(input.get('per_page'), 15), 100);
  return {
    page,
    perPage,
    searchParams: new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
    }),
  };
}

function positiveInteger(value: string | null, fallback: number): number {
  if (!value || !/^[1-9]\d*$/.test(value)) return fallback;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : fallback;
}

function parseOrganizationId(value: string): number | null {
  const result = organizationRouteIdSchema.safeParse(value);
  return result.success ? result.data : null;
}

function unavailable(message: string): OrganizationRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
    }),
  };
}

function unauthenticated(headers: HeadersInit = noStoreHeaders()): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
    headers,
  });
}

function organizationNotFound(headers: HeadersInit = noStoreHeaders()): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
    message: 'Organization not found',
    headers,
  });
}

function noStoreHeaders(): Headers {
  return privateNoStoreHeaders(undefined, ['Cookie']);
}

function contextHeaders(): Headers {
  return privateNoStoreHeaders(undefined, ['Cookie', 'Organization-Id']);
}
