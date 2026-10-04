import type { z } from 'zod';
import { assertContract, type Accepts, type ContractData, type ContractResponse } from '@/http/contract';
import { http } from '@/http/client';
import {
  operatorMemberPageSchema,
  operatorOrganizationPageSchema,
  operatorOrganizationSchema,
  type OperatorMemberPage,
  type OperatorOrganization,
  type OperatorOrganizationPage,
  type OrganizationSearch,
} from '@/features/organizations/types';

export const organizationsService = {
  list(search: OrganizationSearch, signal?: AbortSignal): Promise<OperatorOrganizationPage> {
    const params = new URLSearchParams({ page: String(search.page ?? 1), per_page: '20' });
    if (search.q) {
      params.set('q', search.q);
    }
    return http.get(`/v1/operator/organizations?${params.toString()}`, {
      responseMode: 'json',
      schema: operatorOrganizationPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
  get(organizationId: number, signal?: AbortSignal): Promise<OperatorOrganization> {
    return http.get(`/v1/operator/organizations/${organizationId}`, {
      schema: operatorOrganizationSchema,
      ...(signal ? { signal } : {}),
    });
  },
  members(organizationId: number, page: number, signal?: AbortSignal): Promise<OperatorMemberPage> {
    const params = new URLSearchParams({ page: String(page), per_page: '20' });
    return http.get(`/v1/operator/organizations/${organizationId}/members?${params.toString()}`, {
      responseMode: 'json',
      schema: operatorMemberPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
};

// Fails type-check when contracts/openapi.yaml allows a body these schemas would reject.
assertContract<Accepts<z.input<typeof operatorOrganizationPageSchema>, ContractResponse<'listOperatorOrganizations', 200>>>();
assertContract<Accepts<z.input<typeof operatorOrganizationSchema>, ContractData<'getOperatorOrganization', 200>>>();
assertContract<Accepts<z.input<typeof operatorMemberPageSchema>, ContractResponse<'listOperatorOrganizationMembers', 200>>>();
