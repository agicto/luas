import { describe, expect, it } from 'vitest';

import { ClientErrorCode } from '@/http/codes';
import {
  parseAccessRole,
  parseMemberAccessRoles,
  parsePermissionContext,
} from '@/features/permission/services/permission-service';

const timestamp = '2026-07-15T10:00:00Z';

describe('permission browser contract validation', () => {
  it('accepts canonical sorted permission resources', () => {
    expect(parsePermissionContext({
      organization_id: 1,
      membership_id: 2,
      is_owner: false,
      access_role_ids: [3],
      permissions: ['projects.read'],
    })).toMatchObject({ membership_id: 2, permissions: ['projects.read'] });

    expect(parseAccessRole({
      id: 3,
      organization_id: 1,
      name: 'Project viewer',
      slug: 'project-viewer',
      permissions: ['projects.read'],
      created_at: timestamp,
      updated_at: timestamp,
    })).toMatchObject({ id: 3, slug: 'project-viewer' });

    expect(parseMemberAccessRoles({
      member_id: 2,
      access_role_ids: [],
    })).toEqual({ member_id: 2, access_role_ids: [] });
  });

  it.each([
    {
      organization_id: 1,
      membership_id: 2,
      is_owner: false,
      access_role_ids: [3, 3],
      permissions: [],
    },
    {
      organization_id: 1,
      membership_id: 2,
      is_owner: false,
      access_role_ids: [],
      permissions: ['projects.*'],
    },
    {
      organization_id: 1,
      membership_id: 2,
      is_owner: false,
      access_role_ids: [],
      permissions: [],
      leaked_claim: true,
    },
  ])('rejects malformed or expanded permission contexts', (value) => {
    expect(() => parsePermissionContext(value)).toThrowError(
      expect.objectContaining({ errorCode: ClientErrorCode.INVALID_RESPONSE })
    );
  });
});
