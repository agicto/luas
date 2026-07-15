import { describe, expect, it } from 'vitest';

import { PermissionKey } from '@/features/permission/constants';
import { MockPermissionStore } from '@/features/permission/server/mock-permission-store';

describe('mock permission policy parity', () => {
  it('applies owner bypass and exact grants', () => {
    const store = new MockPermissionStore(() => new Date('2026-07-15T10:00:00Z'));

    expect(store.effective(owner(), 1)).toMatchObject({
      is_owner: true,
      permissions: expect.arrayContaining(Object.values(PermissionKey)),
    });
    expect(store.effective(operator(), 1)).toMatchObject({
      is_owner: false,
      permissions: [PermissionKey.ASSIGNMENTS_READ, PermissionKey.ROLES_READ],
    });
    expect(store.create(operator(), 1, {
      name: 'Denied manager',
      slug: 'denied-manager',
      permissions: [PermissionKey.ROLES_READ],
    })).toBe('permission_denied');
  });

  it('rejects unknown keys, duplicate slugs, and cross-organization role IDs', () => {
    const store = new MockPermissionStore(() => new Date('2026-07-15T10:00:00Z'));
    expect(store.create(owner(), 1, {
      name: 'Unknown role',
      slug: 'unknown-role',
      permissions: ['projects.read'],
    })).toBe('permission_unknown');

    const role = store.create(owner(), 1, {
      name: 'Access manager',
      slug: 'access-manager',
      permissions: [PermissionKey.ROLES_READ],
    });
    expect(typeof role).not.toBe('string');
    expect(store.create(owner(), 1, {
      name: 'Duplicate access manager',
      slug: 'access-manager',
      permissions: [],
    })).toBe('role_slug_conflict');
    expect(store.replaceMemberRoles(owner(), 1, 3, {
      access_role_ids: [999],
    })).toBe('role_not_found');
  });
});

function owner() {
  return { id: 'demo-admin', email: 'admin@example.com', name: 'Admin User' };
}

function operator() {
  return { id: 'demo-operator', email: 'operator@example.com', name: 'Morgan Lee' };
}
