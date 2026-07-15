import type { infer as Infer } from 'zod/mini';

import type { PaginationLinks, PaginationMeta } from '@/features/organization/types';
import type {
  accessRoleSchema,
  createAccessRoleSchema,
  memberAccessRolesSchema,
  permissionCatalogSchema,
  permissionContextSchema,
  replaceMemberAccessRolesSchema,
  updateAccessRoleSchema,
} from './schemas';

export type PermissionContext = Infer<typeof permissionContextSchema>;
export type PermissionCatalog = Infer<typeof permissionCatalogSchema>;
export type AccessRole = Infer<typeof accessRoleSchema>;
export type MemberAccessRoles = Infer<typeof memberAccessRolesSchema>;
export type CreateAccessRoleInput = Infer<typeof createAccessRoleSchema>;
export type UpdateAccessRoleInput = Infer<typeof updateAccessRoleSchema>;
export type ReplaceMemberAccessRolesInput = Infer<
  typeof replaceMemberAccessRolesSchema
>;

export interface AccessRolePage {
  items: AccessRole[];
  meta: PaginationMeta;
  links: PaginationLinks;
}
