import type { z } from 'zod';
import { assertContract, type Accepts, type ContractData, type ContractResponse } from '@/http/contract';
import { http } from '@/http/client';
import {
  managedUserPageSchema,
  managedUserSchema,
  type ManagedUser,
  type ManagedUserPage,
  type UserSearch,
} from '@/features/users/types';

export const usersService = {
  list(search: UserSearch, signal?: AbortSignal): Promise<ManagedUserPage> {
    const params = new URLSearchParams({ page: String(search.page ?? 1), per_page: '20' });
    if (search.q) {
      params.set('q', search.q);
    }
    if (search.status && search.status !== 'all') {
      params.set('status', search.status);
    }
    return http.get(`/v1/operator/users?${params.toString()}`, {
      responseMode: 'json',
      schema: managedUserPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
  disable(id: number): Promise<ManagedUser> {
    return http.post(`/v1/operator/users/${id}/disable`, undefined, { schema: managedUserSchema });
  },
  enable(id: number): Promise<ManagedUser> {
    return http.post(`/v1/operator/users/${id}/enable`, undefined, { schema: managedUserSchema });
  },
  revokeSessions(id: number): Promise<void> {
    return http.post(`/v1/operator/users/${id}/sessions/revoke`, undefined, {
      responseMode: 'json',
    });
  },
};

// Fails type-check when contracts/openapi.yaml allows a body these schemas would reject.
assertContract<Accepts<z.input<typeof managedUserPageSchema>, ContractResponse<'listOperatorUsers', 200>>>();
assertContract<Accepts<z.input<typeof managedUserSchema>, ContractData<'disableOperatorUser', 200>>>();
assertContract<Accepts<z.input<typeof managedUserSchema>, ContractData<'enableOperatorUser', 200>>>();
