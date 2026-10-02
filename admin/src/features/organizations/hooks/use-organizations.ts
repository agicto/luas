import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { organizationsService } from '@/features/organizations/services/organizations-service';
import type { OrganizationSearch } from '@/features/organizations/types';

export const organizationsQueryKey = ['operator', 'organizations'] as const;

export function useOrganizations(search: OrganizationSearch) {
  return useQuery({
    queryKey: [...organizationsQueryKey, 'list', search],
    queryFn: ({ signal }) => organizationsService.list(search, signal),
    placeholderData: keepPreviousData,
  });
}

export function useOrganization(organizationId: number) {
  return useQuery({
    queryKey: [...organizationsQueryKey, organizationId],
    queryFn: ({ signal }) => organizationsService.get(organizationId, signal),
  });
}

export function useOrganizationMembers(organizationId: number, page: number) {
  return useQuery({
    queryKey: [...organizationsQueryKey, organizationId, 'members', page],
    queryFn: ({ signal }) => organizationsService.members(organizationId, page, signal),
    placeholderData: keepPreviousData,
  });
}
