import request from '@/http';
import type {
  AccessPermission,
  AccessRole,
  CreateAccessRoleRequest,
  PaginatedAccessRoleResponse,
  UpdateAccessRoleRequest,
} from '@/features/access/types';

const teamRolesPath = (teamId: number) => `/teams/${teamId}/roles`;

export const accessService = {
  permissions: () => request.get<AccessPermission[]>('/permissions'),

  listRoles: (teamId: number, params?: { page?: number; page_size?: number }) =>
    request.get<PaginatedAccessRoleResponse>(teamRolesPath(teamId), { params }),

  getRole: (teamId: number, id: number) =>
    request.get<AccessRole>(`${teamRolesPath(teamId)}/${id}`),

  createRole: (teamId: number, data: CreateAccessRoleRequest) =>
    request.post<AccessRole, CreateAccessRoleRequest>(teamRolesPath(teamId), data),

  updateRole: (teamId: number, id: number, data: UpdateAccessRoleRequest) =>
    request.put<AccessRole, UpdateAccessRoleRequest>(`${teamRolesPath(teamId)}/${id}`, data),

  deleteRole: (teamId: number, id: number) =>
    request.delete<void>(`${teamRolesPath(teamId)}/${id}`),
};

export type AccessService = typeof accessService;
