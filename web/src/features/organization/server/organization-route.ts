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
import type { AuthUser } from '@/features/auth/types';
import {
  createOrganizationSchema,
  organizationRouteIdSchema,
  updateOrganizationSchema,
} from '@/features/organization/schemas';
import {
  mockOrganizationStore,
  type MockOrganizationStoreError,
} from './mock-organization-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import {
  privateNoStoreHeaders,
  privateNoStoreResponse,
} from '@/server/http/private-response';

type OrganizationBackend = 'go-api' | 'mock';

export type AuthenticatedOrganizationBackend =
  | { backend: 'go-api'; accessToken: string }
  | { backend: 'mock'; user: AuthUser };

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

  const result = mockOrganizationStore.list(authentication.user, page, perPage);
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
  const organization = mockOrganizationStore.create(authentication.user, parsed.data);
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

  const organization = mockOrganizationStore.get(authentication.user, organizationId);
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
  const organization = mockOrganizationStore.update(
    authentication.user,
    organizationId,
    parsed.data
  );
  return typeof organization === 'string'
    ? mockOrganizationErrorResponse(organization)
    : apiSuccessResponse(organization, { headers: noStoreHeaders() });
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

  const context = mockOrganizationStore.resolveContext(
    authentication.user,
    organizationId
  );
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

export async function authenticateOrganizationBackend(
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

  const user = await getSessionUser();
  return user
    ? { authenticated: true, backend, user }
    : { authenticated: false, response: unauthenticated() };
}

export function paginationFromRequest(request: Request) {
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

export function parseOrganizationId(value: string): number | null {
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

export function organizationNotFound(
  headers: HeadersInit = noStoreHeaders()
): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
    message: 'Organization not found',
    headers,
  });
}

export function noStoreHeaders(): Headers {
  return privateNoStoreHeaders(undefined, ['Cookie']);
}

function contextHeaders(): Headers {
  return privateNoStoreHeaders(undefined, ['Cookie', 'Organization-Id']);
}

export function mockOrganizationErrorResponse(
  error: MockOrganizationStoreError
): NextResponse {
  const response = mockOrganizationError(error);
  return apiErrorResponse({ ...response, headers: noStoreHeaders() });
}

function mockOrganizationError(error: MockOrganizationStoreError): {
  status: number;
  errorCode: (typeof ApiErrorCode)[keyof typeof ApiErrorCode];
  message: string;
} {
  switch (error) {
    case 'organization_not_found':
      return {
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
        message: 'Organization not found',
      };
    case 'member_not_found':
      return {
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_MEMBER_NOT_FOUND,
        message: 'Organization member not found',
      };
    case 'invitation_not_found':
      return {
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_INVITATION_NOT_FOUND,
        message: 'Organization invitation not found',
      };
    case 'invitation_invalid':
      return {
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_INVITATION_INVALID,
        message: 'Organization invitation is invalid',
      };
    case 'invitation_expired':
      return {
        status: 410,
        errorCode: ApiErrorCode.ORGANIZATION_INVITATION_EXPIRED,
        message: 'Organization invitation has expired',
      };
    case 'invitation_email_mismatch':
      return {
        status: 403,
        errorCode: ApiErrorCode.ORGANIZATION_INVITATION_EMAIL_MISMATCH,
        message: 'Organization invitation belongs to another account',
      };
    case 'permission_denied':
      return {
        status: 403,
        errorCode: ApiErrorCode.PERMISSION_DENIED,
        message: 'Operation forbidden',
      };
    case 'slug_conflict':
      return {
        status: 409,
        errorCode: ApiErrorCode.ORGANIZATION_SLUG_ALREADY_EXISTS,
        message: 'Organization slug already exists',
      };
    case 'invitation_already_pending':
      return {
        status: 409,
        errorCode: ApiErrorCode.ORGANIZATION_INVITATION_ALREADY_PENDING,
        message: 'Organization invitation is already pending',
      };
    case 'member_already_exists':
      return {
        status: 409,
        errorCode: ApiErrorCode.ORGANIZATION_MEMBER_ALREADY_EXISTS,
        message: 'Organization member already exists',
      };
    case 'ownership_transfer_required':
      return {
        status: 409,
        errorCode: ApiErrorCode.ORGANIZATION_OWNERSHIP_TRANSFER_REQUIRED,
        message: 'Organization ownership transfer is required',
      };
    case 'ownership_transfer_target_invalid':
      return {
        status: 409,
        errorCode: ApiErrorCode.ORGANIZATION_OWNERSHIP_TRANSFER_TARGET_INVALID,
        message: 'Organization ownership transfer target is invalid',
      };
  }
}
