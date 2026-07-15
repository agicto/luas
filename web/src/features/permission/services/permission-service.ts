import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  accessRolePageEnvelopeSchema,
  accessRoleSchema,
  memberAccessRolesSchema,
  permissionCatalogSchema,
  permissionContextSchema,
} from '@/features/permission/schemas';
import type {
  AccessRole,
  AccessRolePage,
  CreateAccessRoleInput,
  MemberAccessRoles,
  PermissionCatalog,
  PermissionContext,
  ReplaceMemberAccessRolesInput,
  UpdateAccessRoleInput,
} from '@/features/permission/types';

interface PageOptions {
  page?: number;
  perPage?: number;
}

export const permissionService = {
  async effective(organizationId: number): Promise<PermissionContext> {
    const value = await request.get<unknown>('/permission-context', context(organizationId));
    return parsePermissionContext(value);
  },

  async catalog(organizationId: number): Promise<PermissionCatalog> {
    const value = await request.get<unknown>('/permissions', context(organizationId));
    return parsePermissionCatalog(value);
  },

  async listRoles(
    organizationId: number,
    { page = 1, perPage = 100 }: PageOptions = {}
  ): Promise<AccessRolePage> {
    const value = await request.getEnvelope<unknown>('/access-roles', {
      ...context(organizationId),
      params: { page, per_page: perPage },
    });
    return parseAccessRolePage(value);
  },

  async getRole(organizationId: number, roleId: number): Promise<AccessRole> {
    const value = await request.get<unknown>(
      `/access-roles/${roleId}`,
      context(organizationId)
    );
    return parseAccessRole(value);
  },

  async createRole(
    organizationId: number,
    input: CreateAccessRoleInput
  ): Promise<AccessRole> {
    const value = await request.post<unknown, CreateAccessRoleInput>(
      '/access-roles',
      input,
      context(organizationId)
    );
    return parseAccessRole(value);
  },

  async updateRole(
    organizationId: number,
    roleId: number,
    input: UpdateAccessRoleInput
  ): Promise<AccessRole> {
    const value = await request.patch<unknown, UpdateAccessRoleInput>(
      `/access-roles/${roleId}`,
      input,
      context(organizationId)
    );
    return parseAccessRole(value);
  },

  async deleteRole(organizationId: number, roleId: number): Promise<void> {
    await request.delete<void>(`/access-roles/${roleId}`, context(organizationId));
  },

  async memberRoles(
    organizationId: number,
    memberId: number
  ): Promise<MemberAccessRoles> {
    const value = await request.get<unknown>(
      `/organization-members/${memberId}/access-roles`,
      context(organizationId)
    );
    return parseMemberAccessRoles(value);
  },

  async replaceMemberRoles(
    organizationId: number,
    memberId: number,
    input: ReplaceMemberAccessRolesInput
  ): Promise<MemberAccessRoles> {
    const value = await request.put<unknown, ReplaceMemberAccessRolesInput>(
      `/organization-members/${memberId}/access-roles`,
      input,
      context(organizationId)
    );
    return parseMemberAccessRoles(value);
  },
};

export function parsePermissionContext(value: unknown): PermissionContext {
  const parsed = permissionContextSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parsePermissionCatalog(value: unknown): PermissionCatalog {
  const parsed = permissionCatalogSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseAccessRolePage(value: unknown): AccessRolePage {
  const parsed = accessRolePageEnvelopeSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return {
    items: parsed.data.data,
    meta: parsed.data.meta,
    links: parsed.data.links,
  };
}

export function parseAccessRole(value: unknown): AccessRole {
  const parsed = accessRoleSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseMemberAccessRoles(value: unknown): MemberAccessRoles {
  const parsed = memberAccessRolesSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function context(organizationId: number) {
  return { headers: { 'Organization-Id': String(organizationId) } };
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Permission service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}
