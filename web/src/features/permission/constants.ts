export const PermissionKey = {
  ROLES_READ: 'permission.roles.read',
  ROLES_MANAGE: 'permission.roles.manage',
  ASSIGNMENTS_READ: 'permission.assignments.read',
  ASSIGNMENTS_MANAGE: 'permission.assignments.manage',
} as const;

export const DEFAULT_PERMISSION_CATALOG: readonly string[] = Object.values(PermissionKey).sort();
