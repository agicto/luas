import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  organizationContextSchema,
  organizationInvitationCreateResultSchema,
  organizationInvitationPageEnvelopeSchema,
  organizationMemberPageEnvelopeSchema,
  organizationMemberSchema,
  organizationOwnershipTransferSchema,
  organizationPageEnvelopeSchema,
  organizationSchema,
} from '@/features/organization/schemas';
import type {
  AcceptOrganizationInvitationInput,
  CreateOrganizationInput,
  CreateOrganizationInvitationInput,
  Organization,
  OrganizationContext,
  OrganizationInvitationCreateResult,
  OrganizationInvitationPage,
  OrganizationMember,
  OrganizationMemberPage,
  OrganizationOwnershipTransfer,
  OrganizationPage,
  TransferOrganizationOwnershipInput,
  UpdateOrganizationMemberInput,
  UpdateOrganizationInput,
} from '@/features/organization/types';

interface PageOptions {
  page?: number;
  perPage?: number;
}

export const organizationService = {
  async list({ page = 1, perPage = 100 }: PageOptions = {}): Promise<OrganizationPage> {
    const value = await request.getEnvelope<unknown>('/organizations', {
      params: { page, per_page: perPage },
    });
    return parseOrganizationPageResponse(value);
  },

  async get(organizationId: number): Promise<Organization> {
    const value = await request.get<unknown>(`/organizations/${organizationId}`);
    return parseOrganizationResponse(value);
  },

  async create(input: CreateOrganizationInput): Promise<Organization> {
    const value = await request.post<unknown, CreateOrganizationInput>(
      '/organizations',
      input
    );
    return parseOrganizationResponse(value);
  },

  async update(
    organizationId: number,
    input: UpdateOrganizationInput
  ): Promise<Organization> {
    const value = await request.patch<unknown, UpdateOrganizationInput>(
      `/organizations/${organizationId}`,
      input
    );
    return parseOrganizationResponse(value);
  },

  async resolveContext(organizationId: number): Promise<OrganizationContext> {
    const value = await request.get<unknown>('/organization-context', {
      headers: { 'Organization-Id': String(organizationId) },
    });
    return parseOrganizationContextResponse(value);
  },

  async listMembers(
    organizationId: number,
    { page = 1, perPage = 100 }: PageOptions = {}
  ): Promise<OrganizationMemberPage> {
    const value = await request.getEnvelope<unknown>(
      `/organizations/${organizationId}/members`,
      { params: { page, per_page: perPage } }
    );
    return parseOrganizationMemberPageResponse(value);
  },

  async changeMemberRole(
    organizationId: number,
    memberId: number,
    input: UpdateOrganizationMemberInput
  ): Promise<OrganizationMember> {
    const value = await request.patch<unknown, UpdateOrganizationMemberInput>(
      `/organizations/${organizationId}/members/${memberId}`,
      input
    );
    return parseOrganizationMemberResponse(value);
  },

  async removeMember(organizationId: number, memberId: number): Promise<void> {
    await request.delete<void>(
      `/organizations/${organizationId}/members/${memberId}`
    );
  },

  async transferOwnership(
    organizationId: number,
    input: TransferOrganizationOwnershipInput
  ): Promise<OrganizationOwnershipTransfer> {
    const value = await request.post<unknown, TransferOrganizationOwnershipInput>(
      `/organizations/${organizationId}/ownership-transfer`,
      input
    );
    return parseOrganizationOwnershipTransferResponse(value);
  },

  async listInvitations(
    organizationId: number,
    { page = 1, perPage = 100 }: PageOptions = {}
  ): Promise<OrganizationInvitationPage> {
    const value = await request.getEnvelope<unknown>(
      `/organizations/${organizationId}/invitations`,
      { params: { page, per_page: perPage } }
    );
    return parseOrganizationInvitationPageResponse(value);
  },

  async invite(
    organizationId: number,
    input: CreateOrganizationInvitationInput
  ): Promise<OrganizationInvitationCreateResult> {
    const value = await request.post<unknown, CreateOrganizationInvitationInput>(
      `/organizations/${organizationId}/invitations`,
      input
    );
    return parseOrganizationInvitationCreateResponse(value);
  },

  async revokeInvitation(
    organizationId: number,
    invitationId: number
  ): Promise<void> {
    await request.delete<void>(
      `/organizations/${organizationId}/invitations/${invitationId}`
    );
  },

  async acceptInvitation(
    input: AcceptOrganizationInvitationInput
  ): Promise<Organization> {
    const value = await request.post<unknown, AcceptOrganizationInvitationInput>(
      '/organization-invitations/accept',
      input
    );
    return parseOrganizationResponse(value);
  },
};

export function parseOrganizationPageResponse(value: unknown): OrganizationPage {
  const envelope = organizationPageEnvelopeSchema.safeParse(value);
  if (!envelope.success) {
    throw invalidResponse();
  }
  return {
    items: envelope.data.data,
    meta: envelope.data.meta,
    links: envelope.data.links,
  };
}

export function parseOrganizationResponse(value: unknown): Organization {
  const parsed = organizationSchema.safeParse(value);
  if (!parsed.success) {
    throw invalidResponse();
  }
  return parsed.data;
}

export function parseOrganizationContextResponse(value: unknown): OrganizationContext {
  const parsed = organizationContextSchema.safeParse(value);
  if (!parsed.success) {
    throw invalidResponse();
  }
  return parsed.data;
}

export function parseOrganizationMemberPageResponse(
  value: unknown
): OrganizationMemberPage {
  const envelope = organizationMemberPageEnvelopeSchema.safeParse(value);
  if (!envelope.success) {
    throw invalidResponse();
  }
  return {
    items: envelope.data.data,
    meta: envelope.data.meta,
    links: envelope.data.links,
  };
}

export function parseOrganizationMemberResponse(value: unknown): OrganizationMember {
  const parsed = organizationMemberSchema.safeParse(value);
  if (!parsed.success) {
    throw invalidResponse();
  }
  return parsed.data;
}

export function parseOrganizationInvitationPageResponse(
  value: unknown
): OrganizationInvitationPage {
  const envelope = organizationInvitationPageEnvelopeSchema.safeParse(value);
  if (!envelope.success) {
    throw invalidResponse();
  }
  return {
    items: envelope.data.data,
    meta: envelope.data.meta,
    links: envelope.data.links,
  };
}

export function parseOrganizationInvitationCreateResponse(
  value: unknown
): OrganizationInvitationCreateResult {
  const parsed = organizationInvitationCreateResultSchema.safeParse(value);
  if (!parsed.success) {
    throw invalidResponse();
  }
  return parsed.data;
}

export function parseOrganizationOwnershipTransferResponse(
  value: unknown
): OrganizationOwnershipTransfer {
  const parsed = organizationOwnershipTransferSchema.safeParse(value);
  if (!parsed.success) {
    throw invalidResponse();
  }
  return parsed.data;
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Organization service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}
