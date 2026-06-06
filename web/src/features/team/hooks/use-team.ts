'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { teamService } from '@/features/team/services/team-service';
import type { CreateTeamRequest, UpdateTeamRequest } from '@/features/team/types';

export const teamQueryKeys = {
  all: ['team'] as const,
  list: (page?: number) => [...teamQueryKeys.all, 'list', page] as const,
  detail: (id: number) => [...teamQueryKeys.all, 'detail', id] as const,
};

export function useTeamList(page = 1) {
  return useQuery({
    queryKey: teamQueryKeys.list(page),
    queryFn: () => teamService.list({ page }),
  });
}

export function useTeam(id: number) {
  return useQuery({
    queryKey: teamQueryKeys.detail(id),
    queryFn: () => teamService.get(id),
    enabled: id > 0,
  });
}

export function useCreateTeam() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTeamRequest) => teamService.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: teamQueryKeys.all }),
  });
}

export function useUpdateTeam(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateTeamRequest) => teamService.update(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: teamQueryKeys.all });
      void queryClient.invalidateQueries({ queryKey: teamQueryKeys.detail(id) });
    },
  });
}

export function useDeleteTeam() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => teamService.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: teamQueryKeys.all }),
  });
}
