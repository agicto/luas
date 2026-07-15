import 'server-only';

import type { NextResponse } from 'next/server';

import {
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
import {
  acceptOrganizationInvitationSchema,
  createOrganizationInvitationSchema,
  transferOrganizationOwnershipSchema,
  updateOrganizationMemberSchema,
} from '@/features/organization/schemas';
import { mockOrganizationStore } from './mock-organization-store';
import {
  authenticateOrganizationBackend,
  mockOrganizationErrorResponse,
  noStoreHeaders,
  paginationFromRequest,
  parseOrganizationId,
  resolveOrganizationRoute,
} from './organization-route';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';

export async function listOrganizationMembersRoute(
  request: Request,
  rawOrganizationId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return mockOrganizationErrorResponse('organization_not_found');
  const { page, perPage, searchParams } = paginationFromRequest(request);

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `organizations/${organizationId}/members`,
      accessToken: authentication.accessToken,
      searchParams,
    });
  }

  const result = mockOrganizationStore.listMembers(
    authentication.user,
    organizationId,
    page,
    perPage
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiPaginatedResponse(result.items, result.meta, result.links, noStoreHeaders());
}

export async function updateOrganizationMemberRoute(
  request: Request,
  rawOrganizationId: string,
  rawMemberId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const ids = memberRouteIds(rawOrganizationId, rawMemberId);
  if (!ids) return mockOrganizationErrorResponse('member_not_found');

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = updateOrganizationMemberSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse(
      'Invalid organization member role payload',
      parsed.error
    );
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `organizations/${ids.organizationId}/members/${ids.memberId}`,
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { role: 'role' },
    });
  }

  const result = mockOrganizationStore.changeMemberRole(
    authentication.user,
    ids.organizationId,
    ids.memberId,
    parsed.data
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiSuccessResponse(result, { headers: noStoreHeaders() });
}

export async function removeOrganizationMemberRoute(
  request: Request,
  rawOrganizationId: string,
  rawMemberId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const ids = memberRouteIds(rawOrganizationId, rawMemberId);
  if (!ids) return mockOrganizationErrorResponse('member_not_found');

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `organizations/${ids.organizationId}/members/${ids.memberId}`,
      accessToken: authentication.accessToken,
    });
  }

  const result = mockOrganizationStore.removeMember(
    authentication.user,
    ids.organizationId,
    ids.memberId
  );
  return result === true
    ? apiNoContentResponse(noStoreHeaders())
    : mockOrganizationErrorResponse(result);
}

export async function transferOrganizationOwnershipRoute(
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
  if (!organizationId) return mockOrganizationErrorResponse('organization_not_found');

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = transferOrganizationOwnershipSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse(
      'Invalid organization ownership transfer payload',
      parsed.error
    );
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: `organizations/${organizationId}/ownership-transfer`,
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { new_owner_member_id: 'new_owner_member_id' },
    });
  }

  const result = mockOrganizationStore.transferOwnership(
    authentication.user,
    organizationId,
    parsed.data
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiSuccessResponse(result, { headers: noStoreHeaders() });
}

export async function listOrganizationInvitationsRoute(
  request: Request,
  rawOrganizationId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return mockOrganizationErrorResponse('organization_not_found');
  const { page, perPage, searchParams } = paginationFromRequest(request);

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `organizations/${organizationId}/invitations`,
      accessToken: authentication.accessToken,
      searchParams,
    });
  }

  const result = mockOrganizationStore.listInvitations(
    authentication.user,
    organizationId,
    page,
    perPage
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiPaginatedResponse(result.items, result.meta, result.links, noStoreHeaders());
}

export async function createOrganizationInvitationRoute(
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
  if (!organizationId) return mockOrganizationErrorResponse('organization_not_found');

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = createOrganizationInvitationSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse(
      'Invalid organization invitation payload',
      parsed.error
    );
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: `organizations/${organizationId}/invitations`,
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { email: 'email', role: 'role' },
    });
  }

  const result = mockOrganizationStore.invite(
    authentication.user,
    organizationId,
    parsed.data
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiSuccessResponse(result, {
        status: 201,
        message: 'created',
        headers: noStoreHeaders(),
      });
}

export async function revokeOrganizationInvitationRoute(
  request: Request,
  rawOrganizationId: string,
  rawInvitationId: string
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;
  const organizationId = parseOrganizationId(rawOrganizationId);
  const invitationId = parseOrganizationId(rawInvitationId);
  if (!organizationId || !invitationId) {
    return mockOrganizationErrorResponse('invitation_not_found');
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `organizations/${organizationId}/invitations/${invitationId}`,
      accessToken: authentication.accessToken,
    });
  }

  const result = mockOrganizationStore.revokeInvitation(
    authentication.user,
    organizationId,
    invitationId
  );
  return result === true
    ? apiNoContentResponse(noStoreHeaders())
    : mockOrganizationErrorResponse(result);
}

export async function acceptOrganizationInvitationRoute(
  request: Request
): Promise<NextResponse> {
  const resolution = resolveOrganizationRoute();
  if (!resolution.available) return resolution.response;
  const guard = guardSameOriginMutation(request);
  if (guard) return guard;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) return authentication.response;

  const payload = await readJsonBody(request);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = acceptOrganizationInvitationSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse(
      'Invalid organization invitation acceptance payload',
      parsed.error
    );
  }

  if (authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'organization-invitations/accept',
      accessToken: authentication.accessToken,
      body: parsed.data,
      fieldMap: { token: 'token' },
    });
  }

  const result = mockOrganizationStore.acceptInvitation(
    authentication.user,
    parsed.data
  );
  return typeof result === 'string'
    ? mockOrganizationErrorResponse(result)
    : apiSuccessResponse(result, { headers: noStoreHeaders() });
}

function memberRouteIds(
  rawOrganizationId: string,
  rawMemberId: string
): { organizationId: number; memberId: number } | null {
  const organizationId = parseOrganizationId(rawOrganizationId);
  const memberId = parseOrganizationId(rawMemberId);
  return organizationId && memberId ? { organizationId, memberId } : null;
}
