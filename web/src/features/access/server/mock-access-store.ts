import type { AccessPermission, AccessRole } from '@/features/access/types';

export const mockAccessPermissions: AccessPermission[] = [
  { key: 'teams:read', label: 'Read teams', category: 'teams' },
  { key: 'teams:update', label: 'Update teams', category: 'teams' },
  { key: 'roles:manage', label: 'Manage roles', category: 'access' },
];

export const mockAccessRoles: AccessRole[] = [
  {
    id: 1,
    team_id: 1,
    name: 'Owner',
    slug: 'owner',
    description: 'Full team access',
    permissions: ['roles:manage', 'teams:read', 'teams:update'],
    system: true,
    created_at: '2026-04-01T00:00:00Z',
    updated_at: '2026-04-01T00:00:00Z',
  },
];
