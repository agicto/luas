import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  organizationContextSchema,
  organizationPageEnvelopeSchema,
  organizationSchema,
} from '@/features/organization/schemas';
import type {
  CreateOrganizationInput,
  Organization,
  OrganizationContext,
  OrganizationPage,
  UpdateOrganizationInput,
} from '@/features/organization/types';

interface OrganizationListOptions {
  page?: number;
  perPage?: number;
}

export const organizationService = {
  async list({ page = 1, perPage = 100 }: OrganizationListOptions = {}): Promise<OrganizationPage> {
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

function invalidResponse(): ApiError {
  return new ApiError(
    'Organization service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}
