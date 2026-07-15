import 'server-only';

import type { AuthUser } from '@/features/auth/types';
import { mockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import {
  DEFAULT_PERMISSION_CATALOG,
  PermissionKey,
} from '@/features/permission/constants';
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

export type MockPermissionStoreError =
  | 'member_not_found'
  | 'organization_not_found'
  | 'permission_denied'
  | 'permission_unknown'
  | 'role_not_found'
  | 'role_slug_conflict';

type RoleRecord = AccessRole;

interface AssignmentRecord {
  organizationId: number;
  membershipId: number;
  roleId: number;
}

export class MockPermissionStore {
  private roles: RoleRecord[];
  private assignments: AssignmentRecord[];
  private nextRoleId = 2;
  private readonly now: () => Date;

  constructor(now: () => Date = () => new Date()) {
    this.now = now;
    const timestamp = now().toISOString();
    this.roles = [
      {
        id: 1,
        organization_id: 1,
        name: 'Access auditor',
        slug: 'access-auditor',
        permissions: [
          PermissionKey.ASSIGNMENTS_READ,
          PermissionKey.ROLES_READ,
        ].sort(),
        created_at: timestamp,
        updated_at: timestamp,
      },
    ];
    this.assignments = [{ organizationId: 1, membershipId: 2, roleId: 1 }];
  }

  effective(
    actor: AuthUser,
    organizationId: number
  ): PermissionContext | 'organization_not_found' {
    const organization = mockOrganizationStore.resolveContext(actor, organizationId);
    if (!organization) return 'organization_not_found';

    const accessRoleIds = this.assignments
      .filter(
        (assignment) =>
          assignment.organizationId === organizationId &&
          assignment.membershipId === organization.membership_id
      )
      .map((assignment) => assignment.roleId)
      .sort((left, right) => left - right);
    const permissions = organization.role === 'owner'
      ? [...DEFAULT_PERMISSION_CATALOG]
      : this.permissionsForRoleIds(organizationId, accessRoleIds);
    return {
      organization_id: organizationId,
      membership_id: organization.membership_id,
      is_owner: organization.role === 'owner',
      access_role_ids: accessRoleIds,
      permissions,
    };
  }

  catalog(
    actor: AuthUser,
    organizationId: number
  ): PermissionCatalog | MockPermissionStoreError {
    const error = this.authorizationError(actor, organizationId, PermissionKey.ROLES_READ);
    return error ? error : { permissions: [...DEFAULT_PERMISSION_CATALOG] };
  }

  list(
    actor: AuthUser,
    organizationId: number,
    page: number,
    perPage: number
  ): AccessRolePage | MockPermissionStoreError {
    const error = this.authorizationError(actor, organizationId, PermissionKey.ROLES_READ);
    if (error) return error;
    const items = this.roles
      .filter((role) => role.organization_id === organizationId)
      .sort((left, right) => left.name.localeCompare(right.name) || left.id - right.id);
    return paginated(items, page, perPage, '/api/access-roles');
  }

  get(
    actor: AuthUser,
    organizationId: number,
    roleId: number
  ): AccessRole | MockPermissionStoreError {
    const error = this.authorizationError(actor, organizationId, PermissionKey.ROLES_READ);
    if (error) return error;
    return this.role(organizationId, roleId) ?? 'role_not_found';
  }

  create(
    actor: AuthUser,
    organizationId: number,
    input: CreateAccessRoleInput
  ): AccessRole | MockPermissionStoreError {
    const effective = this.effective(actor, organizationId);
    if (effective === 'organization_not_found') return effective;
    if (!hasPermission(effective, PermissionKey.ROLES_MANAGE)) return 'permission_denied';
    if (!this.knownPermissions(input.permissions)) return 'permission_unknown';
    if (!dominates(effective, input.permissions)) return 'permission_denied';
    if (
      this.roles.some(
        (role) => role.organization_id === organizationId && role.slug === input.slug
      )
    ) {
      return 'role_slug_conflict';
    }

    const timestamp = this.now().toISOString();
    const role: RoleRecord = {
      id: this.nextRoleId++,
      organization_id: organizationId,
      name: input.name,
      slug: input.slug,
      permissions: [...input.permissions].sort(),
      created_at: timestamp,
      updated_at: timestamp,
    };
    this.roles = [...this.roles, role];
    return cloneRole(role);
  }

  update(
    actor: AuthUser,
    organizationId: number,
    roleId: number,
    input: UpdateAccessRoleInput
  ): AccessRole | MockPermissionStoreError {
    const effective = this.effective(actor, organizationId);
    if (effective === 'organization_not_found') return effective;
    if (!hasPermission(effective, PermissionKey.ROLES_MANAGE)) return 'permission_denied';
    if (!this.knownPermissions(input.permissions)) return 'permission_unknown';
    const role = this.role(organizationId, roleId);
    if (!role) return 'role_not_found';
    if (
      !dominates(effective, role.permissions) ||
      !dominates(effective, input.permissions)
    ) {
      return 'permission_denied';
    }
    role.name = input.name;
    role.permissions = [...input.permissions].sort();
    role.updated_at = this.now().toISOString();
    return cloneRole(role);
  }

  delete(
    actor: AuthUser,
    organizationId: number,
    roleId: number
  ): true | MockPermissionStoreError {
    const effective = this.effective(actor, organizationId);
    if (effective === 'organization_not_found') return effective;
    if (!hasPermission(effective, PermissionKey.ROLES_MANAGE)) return 'permission_denied';
    const role = this.role(organizationId, roleId);
    if (!role) return 'role_not_found';
    if (!dominates(effective, role.permissions)) return 'permission_denied';

    this.roles = this.roles.filter((candidate) => candidate.id !== roleId);
    this.assignments = this.assignments.filter(
      (assignment) => assignment.roleId !== roleId
    );
    return true;
  }

  memberRoles(
    actor: AuthUser,
    organizationId: number,
    memberId: number
  ): MemberAccessRoles | MockPermissionStoreError {
    const error = this.authorizationError(
      actor,
      organizationId,
      PermissionKey.ASSIGNMENTS_READ
    );
    if (error) return error;
    if (!this.memberExists(actor, organizationId, memberId)) return 'member_not_found';
    return {
      member_id: memberId,
      access_role_ids: this.assignmentRoleIds(organizationId, memberId),
    };
  }

  replaceMemberRoles(
    actor: AuthUser,
    organizationId: number,
    memberId: number,
    input: ReplaceMemberAccessRolesInput
  ): MemberAccessRoles | MockPermissionStoreError {
    const effective = this.effective(actor, organizationId);
    if (effective === 'organization_not_found') return effective;
    if (!hasPermission(effective, PermissionKey.ASSIGNMENTS_MANAGE)) {
      return 'permission_denied';
    }
    if (!this.memberExists(actor, organizationId, memberId)) return 'member_not_found';

    const currentRoleIds = this.assignmentRoleIds(organizationId, memberId);
    const currentRoles = currentRoleIds.map((roleId) => this.role(organizationId, roleId));
    const requestedRoles = input.access_role_ids.map((roleId) => this.role(organizationId, roleId));
    if (requestedRoles.some((role) => !role)) return 'role_not_found';
    const touchedPermissions = [...currentRoles, ...requestedRoles].flatMap(
      (role) => role?.permissions ?? []
    );
    if (!dominates(effective, touchedPermissions)) return 'permission_denied';

    this.assignments = this.assignments.filter(
      (assignment) =>
        assignment.organizationId !== organizationId ||
        assignment.membershipId !== memberId
    );
    for (const roleId of input.access_role_ids) {
      this.assignments.push({ organizationId, membershipId: memberId, roleId });
    }
    return {
      member_id: memberId,
      access_role_ids: [...input.access_role_ids].sort((left, right) => left - right),
    };
  }

  private authorizationError(
    actor: AuthUser,
    organizationId: number,
    permission: string
  ): MockPermissionStoreError | null {
    const effective = this.effective(actor, organizationId);
    if (effective === 'organization_not_found') return effective;
    return hasPermission(effective, permission) ? null : 'permission_denied';
  }

  private role(organizationId: number, roleId: number): RoleRecord | undefined {
    return this.roles.find(
      (role) => role.organization_id === organizationId && role.id === roleId
    );
  }

  private assignmentRoleIds(organizationId: number, memberId: number): number[] {
    return this.assignments
      .filter(
        (assignment) =>
          assignment.organizationId === organizationId &&
          assignment.membershipId === memberId
      )
      .map((assignment) => assignment.roleId)
      .sort((left, right) => left - right);
  }

  private permissionsForRoleIds(organizationId: number, roleIds: number[]): string[] {
    return [...new Set(roleIds.flatMap((roleId) => this.role(organizationId, roleId)?.permissions ?? []))].sort();
  }

  private knownPermissions(permissions: string[]): boolean {
    return permissions.every((permission) => DEFAULT_PERMISSION_CATALOG.includes(permission));
  }

  private memberExists(actor: AuthUser, organizationId: number, memberId: number): boolean {
    const page = mockOrganizationStore.listMembers(actor, organizationId, 1, 100);
    return typeof page !== 'string' && page.items.some((member) => member.id === memberId);
  }
}

export const mockPermissionStore = new MockPermissionStore();

function hasPermission(context: PermissionContext, permission: string): boolean {
  return context.is_owner || context.permissions.includes(permission);
}

function dominates(context: PermissionContext, permissions: readonly string[]): boolean {
  return context.is_owner || permissions.every((permission) => context.permissions.includes(permission));
}

function cloneRole(role: RoleRecord): AccessRole {
  return { ...role, permissions: [...role.permissions] };
}

function paginated<T>(
  values: T[],
  page: number,
  perPage: number,
  path: string
): { items: T[]; meta: AccessRolePage['meta']; links: AccessRolePage['links'] } {
  const total = values.length;
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const currentPage = Math.min(page, lastPage);
  const offset = (currentPage - 1) * perPage;
  const items = values.slice(offset, offset + perPage);
  const url = (target: number) => `${path}?page=${target}&per_page=${perPage}`;
  return {
    items,
    meta: {
      current_page: currentPage,
      per_page: perPage,
      total,
      last_page: lastPage,
      from: items.length === 0 ? 0 : offset + 1,
      to: items.length === 0 ? 0 : offset + items.length,
    },
    links: {
      first: url(1),
      last: url(lastPage),
      prev: currentPage > 1 ? url(currentPage - 1) : null,
      next: currentPage < lastPage ? url(currentPage + 1) : null,
    },
  };
}
