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
import { isWebFeatureEnabled } from '@/config/features';
import {
  authenticateOrganizationBackend,
  noStoreHeaders,
  organizationNotFound,
  paginationFromRequest,
  parseOrganizationId,
  resolveOrganizationRoute,
} from '@/features/organization/server/organization-route';
import {
  createAccessRoleSchema,
  replaceMemberAccessRolesSchema,
  updateAccessRoleSchema,
} from '@/features/permission/schemas';
import {
  mockPermissionStore,
  type MockPermissionStoreError,
} from '@/features/permission/server/mock-permission-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';

type PermissionBackend = 'go-api' | 'mock';

type PermissionRouteResolution =
  | { available: true; backend: PermissionBackend }
  | { available: false; response: NextResponse };

export function resolvePermissionRoute(): PermissionRouteResolution {
  if (!isWebFeatureEnabled('permission')) {
    return unavailable('Permission Web feature is disabled');
  }
  const organization = resolveOrganizationRoute();
  return organization.available
    ? { available: true, backend: organization.backend }
    : organization;
}

export async function getPermissionContextRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute(request);
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'permission-context',
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
    });
  }
  const value = mockPermissionStore.effective(
    route.authentication.user,
    route.organizationId
  );
  return value === 'organization_not_found'
    ? organizationNotFound(contextHeaders())
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

export async function listPermissionsRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute(request);
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'permissions',
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
    });
  }
  const value = mockPermissionStore.catalog(
    route.authentication.user,
    route.organizationId
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

export async function listAccessRolesRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute(request);
  if (!route.available) return route.response;
  const { page, perPage, searchParams } = paginationFromRequest(request);
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'access-roles',
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
      searchParams,
    });
  }
  const value = mockPermissionStore.list(
    route.authentication.user,
    route.organizationId,
    page,
    perPage
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiPaginatedResponse(value.items, value.meta, value.links, contextHeaders());
}

export async function createAccessRoleRoute(request: Request): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = createAccessRoleSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid access role payload', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'access-roles',
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
      body: parsed.data,
      fieldMap: { name: 'name', slug: 'slug', permissions: 'permissions' },
    });
  }
  const value = mockPermissionStore.create(
    route.authentication.user,
    route.organizationId,
    parsed.data
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, {
        status: 201,
        message: 'created',
        headers: contextHeaders(),
      });
}

export async function getAccessRoleRoute(
  request: Request,
  rawRoleId: string
): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute(request);
  if (!route.available) return route.response;
  const roleId = parseOrganizationId(rawRoleId);
  if (!roleId) return mockPermissionErrorResponse('role_not_found');
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `access-roles/${roleId}`,
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
    });
  }
  const value = mockPermissionStore.get(
    route.authentication.user,
    route.organizationId,
    roleId
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

export async function updateAccessRoleRoute(
  request: Request,
  rawRoleId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const roleId = parseOrganizationId(rawRoleId);
  if (!roleId) return mockPermissionErrorResponse('role_not_found');
  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = updateAccessRoleSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid access role update', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `access-roles/${roleId}`,
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
      body: parsed.data,
      fieldMap: { name: 'name', permissions: 'permissions' },
    });
  }
  const value = mockPermissionStore.update(
    route.authentication.user,
    route.organizationId,
    roleId,
    parsed.data
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

export async function deleteAccessRoleRoute(
  request: Request,
  rawRoleId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const roleId = parseOrganizationId(rawRoleId);
  if (!roleId) return mockPermissionErrorResponse('role_not_found');
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `access-roles/${roleId}`,
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
    });
  }
  const value = mockPermissionStore.delete(
    route.authentication.user,
    route.organizationId,
    roleId
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiNoContentResponse(contextHeaders());
}

export async function getMemberAccessRolesRoute(
  request: Request,
  rawMemberId: string
): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute(request);
  if (!route.available) return route.response;
  const memberId = parseOrganizationId(rawMemberId);
  if (!memberId) return mockPermissionErrorResponse('member_not_found');
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `organization-members/${memberId}/access-roles`,
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
    });
  }
  const value = mockPermissionStore.memberRoles(
    route.authentication.user,
    route.organizationId,
    memberId
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

export async function replaceMemberAccessRolesRoute(
  request: Request,
  rawMemberId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const memberId = parseOrganizationId(rawMemberId);
  if (!memberId) return mockPermissionErrorResponse('member_not_found');
  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = replaceMemberAccessRolesSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid member access roles', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PUT',
      path: `organization-members/${memberId}/access-roles`,
      accessToken: route.authentication.accessToken,
      organizationId: String(route.organizationId),
      body: parsed.data,
      fieldMap: { access_role_ids: 'access_role_ids' },
    });
  }
  const value = mockPermissionStore.replaceMemberRoles(
    route.authentication.user,
    route.organizationId,
    memberId,
    parsed.data
  );
  return typeof value === 'string'
    ? mockPermissionErrorResponse(value)
    : apiSuccessResponse(value, { headers: contextHeaders() });
}

async function resolveMutationRoute(request: Request) {
  const resolution = resolvePermissionRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  return guard
    ? { available: false as const, response: guard }
    : resolveAuthenticatedRoute(request, resolution);
}

async function resolveAuthenticatedRoute(
  request: Request,
  resolution: PermissionRouteResolution = resolvePermissionRoute()
) {
  if (!resolution.available) return resolution;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) {
    return { available: false as const, response: authentication.response };
  }
  const organizationId = organizationIdFromRequest(request);
  if (!organizationId.ok) return { available: false as const, response: organizationId.response };
  return {
    available: true as const,
    organizationId: organizationId.value,
    authentication,
  };
}

function organizationIdFromRequest(request: Request):
  | { ok: true; value: number }
  | { ok: false; response: NextResponse } {
  const value = request.headers.get('organization-id');
  if (value === null) {
    return {
      ok: false,
      response: apiErrorResponse({
        status: 400,
        errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
        message: 'Organization context is required',
        headers: contextHeaders(),
      }),
    };
  }
  const organizationId = parseOrganizationId(value);
  if (!organizationId) {
    return {
      ok: false,
      response: apiErrorResponse({
        status: 400,
        errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
        message: 'Organization context is invalid',
        headers: contextHeaders(),
      }),
    };
  }
  return { ok: true, value: organizationId };
}

function mockPermissionErrorResponse(error: MockPermissionStoreError): NextResponse {
  const headers = contextHeaders();
  switch (error) {
    case 'organization_not_found':
      return organizationNotFound(headers);
    case 'member_not_found':
      return apiErrorResponse({
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_MEMBER_NOT_FOUND,
        message: 'Organization member not found',
        headers,
      });
    case 'role_not_found':
      return apiErrorResponse({
        status: 404,
        errorCode: ApiErrorCode.PERMISSION_ROLE_NOT_FOUND,
        message: 'Access role not found',
        headers,
      });
    case 'role_slug_conflict':
      return apiErrorResponse({
        status: 409,
        errorCode: ApiErrorCode.PERMISSION_ROLE_SLUG_ALREADY_EXISTS,
        message: 'Access role slug already exists',
        headers,
      });
    case 'permission_unknown':
      return apiErrorResponse({
        status: 422,
        errorCode: ApiErrorCode.PERMISSION_UNKNOWN,
        message: 'Permission is not registered',
        headers,
      });
    case 'permission_denied':
      return apiErrorResponse({
        status: 403,
        errorCode: ApiErrorCode.PERMISSION_DENIED,
        message: 'Permission denied',
        headers,
      });
  }
}

function contextHeaders(): Headers {
  const headers = noStoreHeaders();
  headers.append('Vary', 'Organization-Id');
  return headers;
}

function unavailable(message: string): PermissionRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
    }),
  };
}
