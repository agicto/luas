'use client';

import { useQuery } from '@tanstack/react-query';

import { usageService } from '@/features/usage/services/usage-service';

export const usageKeys = {
  all: ['usage'] as const,
  user: () => [...usageKeys.all, 'user'] as const,
  organization: (organizationId: number) =>
    [...usageKeys.all, 'organization', organizationId] as const,
};

export function useUserUsage() {
  return useQuery({
    queryKey: usageKeys.user(),
    queryFn: () => usageService.user(),
    staleTime: 30_000,
  });
}

export function useOrganizationUsage(organizationId: number) {
  return useQuery({
    queryKey: usageKeys.organization(organizationId),
    queryFn: () => usageService.organization(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 30_000,
  });
}
