'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { accessService } from '@/features/access/services/access-service';
import type {
  CreateAccessRoleRequest,
  UpdateAccessRoleRequest,
} from '@/features/access/types';

export const accessQueryKeys = {
  all: ['access'] as const,
  permissions: () => [...accessQueryKeys.all, 'permissions'] as const,
  roleList: (teamId: number, page?: number) =>
    [...accessQueryKeys.all, 'teams', teamId, 'roles', page] as const,
  roleDetail: (teamId: number, id: number) =>
    [...accessQueryKeys.all, 'teams', teamId, 'roles', id] as const,
};

export function useAccessPermissions() {
  return useQuery({
    queryKey: accessQueryKeys.permissions(),
    queryFn: () => accessService.permissions(),
  });
}

export function useAccessRoleList(teamId: number, page = 1) {
  return useQuery({
    queryKey: accessQueryKeys.roleList(teamId, page),
    queryFn: () => accessService.listRoles(teamId, { page }),
    enabled: teamId > 0,
  });
}

export function useAccessRole(teamId: number, id: number) {
  return useQuery({
    queryKey: accessQueryKeys.roleDetail(teamId, id),
    queryFn: () => accessService.getRole(teamId, id),
    enabled: teamId > 0 && id > 0,
  });
}

export function useCreateAccessRole(teamId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateAccessRoleRequest) => accessService.createRole(teamId, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: accessQueryKeys.all }),
  });
}

export function useUpdateAccessRole(teamId: number, id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateAccessRoleRequest) => accessService.updateRole(teamId, id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: accessQueryKeys.all });
      void queryClient.invalidateQueries({ queryKey: accessQueryKeys.roleDetail(teamId, id) });
    },
  });
}

export function useDeleteAccessRole(teamId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => accessService.deleteRole(teamId, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: accessQueryKeys.all }),
  });
}
