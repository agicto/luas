'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { permissionService } from '@/features/permission/services/permission-service';
import type {
  AccessRole,
  AccessRolePage,
  CreateAccessRoleInput,
  ReplaceMemberAccessRolesInput,
  UpdateAccessRoleInput,
} from '@/features/permission/types';

export const permissionKeys = {
  all: ['permissions'] as const,
  effective: (organizationId: number) =>
    [...permissionKeys.all, organizationId, 'effective'] as const,
  catalog: (organizationId: number) =>
    [...permissionKeys.all, organizationId, 'catalog'] as const,
  roles: (organizationId: number) =>
    [...permissionKeys.all, organizationId, 'roles'] as const,
  memberRoles: (organizationId: number, memberId: number) =>
    [...permissionKeys.all, organizationId, 'members', memberId] as const,
};

export function usePermissionContext(organizationId: number) {
  return useQuery({
    queryKey: permissionKeys.effective(organizationId),
    queryFn: () => permissionService.effective(organizationId),
    enabled: validId(organizationId),
    staleTime: 10_000,
  });
}

export function usePermissionCatalog(organizationId: number, enabled = true) {
  return useQuery({
    queryKey: permissionKeys.catalog(organizationId),
    queryFn: () => permissionService.catalog(organizationId),
    enabled: enabled && validId(organizationId),
    staleTime: 60_000,
  });
}

export function useAccessRoles(organizationId: number, enabled = true) {
  return useQuery({
    queryKey: permissionKeys.roles(organizationId),
    queryFn: () => permissionService.listRoles(organizationId),
    enabled: enabled && validId(organizationId),
    staleTime: 10_000,
  });
}

export function useMemberAccessRoles(
  organizationId: number,
  memberId: number,
  enabled = true
) {
  return useQuery({
    queryKey: permissionKeys.memberRoles(organizationId, memberId),
    queryFn: () => permissionService.memberRoles(organizationId, memberId),
    enabled: enabled && validId(organizationId) && validId(memberId),
    staleTime: 10_000,
  });
}

export function useCreateAccessRole(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateAccessRoleInput) =>
      permissionService.createRole(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (role) => upsertRole(queryClient, organizationId, role),
    onSettled: () => invalidatePermissionState(queryClient, organizationId),
  });
}

export function useUpdateAccessRole(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId, input }: { roleId: number; input: UpdateAccessRoleInput }) =>
      permissionService.updateRole(organizationId, roleId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (role) => upsertRole(queryClient, organizationId, role),
    onSettled: () => invalidatePermissionState(queryClient, organizationId),
  });
}

export function useDeleteAccessRole(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (roleId: number) => permissionService.deleteRole(organizationId, roleId),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (_value, roleId) => {
      queryClient.setQueryData<AccessRolePage>(
        permissionKeys.roles(organizationId),
        (current) => current
          ? {
              ...current,
              items: current.items.filter((role) => role.id !== roleId),
              meta: { ...current.meta, total: Math.max(0, current.meta.total - 1) },
            }
          : current
      );
    },
    onSettled: () => invalidatePermissionState(queryClient, organizationId),
  });
}

export function useReplaceMemberAccessRoles(organizationId: number, memberId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ReplaceMemberAccessRolesInput) =>
      permissionService.replaceMemberRoles(organizationId, memberId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (assignment) => {
      queryClient.setQueryData(
        permissionKeys.memberRoles(organizationId, memberId),
        assignment
      );
    },
    onSettled: () => invalidatePermissionState(queryClient, organizationId),
  });
}

function upsertRole(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: number,
  role: AccessRole
) {
  queryClient.setQueryData<AccessRolePage>(
    permissionKeys.roles(organizationId),
    (current) => current
      ? {
          ...current,
          items: [role, ...current.items.filter((item) => item.id !== role.id)]
            .sort((left, right) => left.name.localeCompare(right.name) || left.id - right.id),
          meta: {
            ...current.meta,
            total: current.items.some((item) => item.id === role.id)
              ? current.meta.total
              : current.meta.total + 1,
          },
        }
      : current
  );
}

function invalidatePermissionState(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: number
) {
  queryClient.invalidateQueries({
    queryKey: [...permissionKeys.all, organizationId],
  });
}

function validId(value: number): boolean {
  return Number.isSafeInteger(value) && value > 0;
}
