'use client';

import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { organizationService } from '@/features/organization/services/organization-service';
import type {
  CreateOrganizationInput,
  Organization,
  OrganizationContext,
  OrganizationPage,
  UpdateOrganizationInput,
} from '@/features/organization/types';

export const organizationKeys = {
  all: ['organizations'] as const,
  list: () => [...organizationKeys.all, 'list'] as const,
  detail: (organizationId: number) =>
    [...organizationKeys.all, 'detail', organizationId] as const,
  context: (organizationId: number) =>
    [...organizationKeys.all, 'context', organizationId] as const,
};

export function useOrganizations(enabled = true) {
  return useQuery({
    queryKey: organizationKeys.list(),
    queryFn: () => organizationService.list(),
    enabled,
    staleTime: 30_000,
  });
}

export function useOrganization(organizationId: number) {
  return useQuery({
    queryKey: organizationKeys.detail(organizationId),
    queryFn: () => organizationService.get(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 30_000,
  });
}

export function useOrganizationContext(organizationId: number) {
  return useQuery({
    queryKey: organizationKeys.context(organizationId),
    queryFn: () => organizationService.resolveContext(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 15_000,
  });
}

export function useCreateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateOrganizationInput) =>
      organizationService.create(input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (organization) => {
      queryClient.setQueryData(
        organizationKeys.detail(organization.id),
        organization
      );
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
    },
  });
}

export function useUpdateOrganization(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateOrganizationInput) =>
      organizationService.update(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onMutate: async (input) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: organizationKeys.list() }),
        queryClient.cancelQueries({ queryKey: organizationKeys.detail(organizationId) }),
        queryClient.cancelQueries({ queryKey: organizationKeys.context(organizationId) }),
      ]);
      const previousList = queryClient.getQueryData<OrganizationPage>(
        organizationKeys.list()
      );
      const previousDetail = queryClient.getQueryData<Organization>(
        organizationKeys.detail(organizationId)
      );
      const previousContext = queryClient.getQueryData<OrganizationContext>(
        organizationKeys.context(organizationId)
      );

      queryClient.setQueryData<OrganizationPage>(organizationKeys.list(), (current) =>
        current
          ? {
              ...current,
              items: current.items.map((organization) =>
                organization.id === organizationId
                  ? { ...organization, name: input.name }
                  : organization
              ),
            }
          : current
      );
      queryClient.setQueryData<Organization>(
        organizationKeys.detail(organizationId),
        (current) => current ? { ...current, name: input.name } : current
      );
      queryClient.setQueryData<OrganizationContext>(
        organizationKeys.context(organizationId),
        (current) => current
          ? { ...current, organization_name: input.name }
          : current
      );

      return { previousList, previousDetail, previousContext };
    },
    onError: (_error, _input, context) => {
      if (!context) return;
      queryClient.setQueryData(organizationKeys.list(), context.previousList);
      queryClient.setQueryData(
        organizationKeys.detail(organizationId),
        context.previousDetail
      );
      queryClient.setQueryData(
        organizationKeys.context(organizationId),
        context.previousContext
      );
    },
    onSuccess: (organization) => {
      queryClient.setQueryData(organizationKeys.detail(organizationId), organization);
      queryClient.setQueryData<OrganizationContext>(
        organizationKeys.context(organizationId),
        (current) => current
          ? { ...current, organization_name: organization.name }
          : current
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
      queryClient.invalidateQueries({ queryKey: organizationKeys.detail(organizationId) });
      queryClient.invalidateQueries({ queryKey: organizationKeys.context(organizationId) });
    },
  });
}
