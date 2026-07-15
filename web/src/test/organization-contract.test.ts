import { describe, expect, it } from 'vitest';

import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import { MockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import {
  acceptOrganizationInvitationSchema,
  createOrganizationInvitationSchema,
  createOrganizationSchema,
} from '@/features/organization/schemas';
import {
  parseOrganizationInvitationCreateResponse,
  parseOrganizationInvitationPageResponse,
  parseOrganizationMemberPageResponse,
  parseOrganizationOwnershipTransferResponse,
  parseOrganizationContextResponse,
  parseOrganizationPageResponse,
  parseOrganizationResponse,
} from '@/features/organization/services/organization-service';
import type { AuthUser } from '@/features/auth/types';

const timestamp = '2026-07-15T10:00:00.000Z';
const organization = {
  id: 42,
  name: 'Acme Europe',
  slug: 'acme-europe',
  role: 'owner',
  created_at: timestamp,
  updated_at: timestamp,
};
const owner: AuthUser = {
  id: 'demo-admin',
  email: 'admin@example.com',
  name: 'Admin User',
};
const member: AuthUser = {
  id: 'demo-member',
  email: 'member@example.com',
  name: 'Riley Chen',
};

describe('organization network contract validation', () => {
  it('counts organization names by Unicode code point like the Go contract', () => {
    const astralCharacter = '\u{1F600}';

    expect(createOrganizationSchema.safeParse({ name: astralCharacter.repeat(100) }).success).toBe(true);
    expect(createOrganizationSchema.safeParse({ name: astralCharacter.repeat(101) }).success).toBe(false);
  });

  it('normalizes names but rejects non-canonical slugs at the browser input boundary', () => {
    expect(createOrganizationSchema.parse({ name: '  Acme Europe  ' })).toEqual({
      name: 'Acme Europe',
    });
    expect(createOrganizationSchema.safeParse({
      name: 'Acme Europe',
      slug: ' acme-europe ',
    }).success).toBe(false);
  });

  it('preserves a valid paginated envelope', () => {
    expect(
      parseOrganizationPageResponse({
        code: 0,
        message: 'success',
        data: [organization],
        meta: {
          current_page: 1,
          per_page: 15,
          total: 1,
          last_page: 1,
          from: 1,
          to: 1,
        },
        links: { first: '/v1/organizations?page=1', last: '', prev: null, next: null },
      })
    ).toMatchObject({
      items: [organization],
      meta: { total: 1 },
    });
  });

  it.each([
    { ...organization, id: Number.MAX_SAFE_INTEGER + 1 },
    { ...organization, name: 'x' },
    { ...organization, name: ' Acme Europe ' },
    { ...organization, slug: 'Not Canonical' },
    { ...organization, role: 'super-admin' },
    { ...organization, updated_at: 'not-a-date' },
    { ...organization, updated_at: '2026-07-15' },
  ])('rejects malformed successful organization data', (value) => {
    expectInvalidResponse(() => parseOrganizationResponse(value));
  });

  it('rejects a context whose IDs are not safe browser integers', () => {
    expectInvalidResponse(() =>
      parseOrganizationContextResponse({
        organization_id: 42,
        organization_name: 'Acme Europe',
        organization_slug: 'acme-europe',
        membership_id: 91,
        user_id: Number.MAX_SAFE_INTEGER + 1,
        role: 'admin',
      })
    );
  });

  it('validates PII-minimized member pages without silently accepting email', () => {
    const memberView = {
      id: 91,
      user_id: 17,
      username: 'alex',
      nickname: 'Alex',
      avatar: 'https://cdn.example.com/alex.png',
      role: 'member',
      joined_at: timestamp,
      updated_at: timestamp,
    };
    expect(
      parseOrganizationMemberPageResponse(
        pageEnvelope([memberView], '/v1/organizations/42/members')
      )
    ).toMatchObject({ items: [memberView], meta: { total: 1 } });

    expectInvalidResponse(() =>
      parseOrganizationMemberPageResponse(
        pageEnvelope(
          [{ ...memberView, email: 'private@example.com' }],
          '/v1/organizations/42/members'
        )
      )
    );
  });

  it('validates token-free invitation history and the separate email attempt status', () => {
    const invitation = {
      id: 73,
      organization_id: 42,
      email: 'member@example.com',
      role: 'member',
      status: 'pending',
      expires_at: timestamp,
      created_at: timestamp,
      updated_at: timestamp,
    };
    expect(
      parseOrganizationInvitationCreateResponse({
        invitation,
        email_send_status: 'not_configured',
      })
    ).toEqual({ invitation, email_send_status: 'not_configured' });
    expect(
      parseOrganizationInvitationPageResponse(
        pageEnvelope([invitation], '/v1/organizations/42/invitations')
      )
    ).toMatchObject({ items: [invitation], meta: { total: 1 } });

    expectInvalidResponse(() =>
      parseOrganizationInvitationCreateResponse({
        invitation: { ...invitation, token: 'must-not-cross-the-contract' },
        email_send_status: 'accepted_by_provider',
      })
    );
  });

  it('validates ownership transfer and normalizes invitation inputs', () => {
    const previousOwner = memberView({ id: 91, user_id: 17, role: 'admin' });
    const newOwner = memberView({ id: 92, user_id: 18, role: 'owner' });

    expect(
      parseOrganizationOwnershipTransferResponse({ previous_owner: previousOwner, new_owner: newOwner })
    ).toEqual({ previous_owner: previousOwner, new_owner: newOwner });
    expect(
      createOrganizationInvitationSchema.parse({
        email: '  NEW.MEMBER@example.com  ',
        role: 'member',
      })
    ).toEqual({ email: 'new.member@example.com', role: 'member' });
    expect(createOrganizationInvitationSchema.safeParse({
      email: 'new.member@example.com',
      role: 'owner',
    }).success).toBe(false);
    expect(acceptOrganizationInvitationSchema.parse({ token: '  one-time-token  ' })).toEqual({
      token: 'one-time-token',
    });
  });
});

describe('MockOrganizationStore', () => {
  it('supports the list, create, resolve, and rename browser workflow', () => {
    const store = new MockOrganizationStore({
      now: () => new Date(timestamp),
      randomUUID: () => '01234567-89ab-cdef-0123-456789abcdef',
    });

    const created = store.create(owner, { name: 'Research Lab' });
    expect(created).toMatchObject({
      id: 2,
      name: 'Research Lab',
      slug: 'org-0123456789abcdef01234567',
      role: 'owner',
    });
    expect(store.list(owner, 1, 15).meta.total).toBe(2);
    expect(store.resolveContext(owner, 2)).toMatchObject({
      organization_id: 2,
      membership_id: 4,
      role: 'owner',
    });
    expect(store.update(owner, 2, { name: 'Research Europe' })).toMatchObject({
      name: 'Research Europe',
      slug: 'org-0123456789abcdef01234567',
    });
    expect(store.list(owner, 1, 7).links).toMatchObject({
      first: '/api/organizations?page=1&per_page=7',
      last: '/api/organizations?page=1&per_page=7',
    });
  });

  it('preserves global mock slug uniqueness', () => {
    const store = new MockOrganizationStore();
    expect(store.create(owner, { name: 'First', slug: 'shared-slug' })).not.toBe('slug_conflict');
    expect(store.create(owner, { name: 'Second', slug: 'shared-slug' })).toBe('slug_conflict');
  });

  it('enforces member roles, ownership transfer, and token-free invitation state', () => {
    const invitationToken = 'oinv_test.one-time-secret';
    const store = new MockOrganizationStore({
      now: () => new Date(timestamp),
      invitationToken: () => invitationToken,
    });

    const directory = store.listMembers(owner, 1, 1, 15);
    expect(directory).toMatchObject({ meta: { total: 3 } });
    expect(JSON.stringify(directory)).not.toContain('admin@example.com');
    expect(store.changeMemberRole(member, 1, 2, { role: 'member' })).toBe(
      'permission_denied'
    );
    expect(store.changeMemberRole(owner, 1, 3, { role: 'admin' })).toMatchObject({
      id: 3,
      role: 'admin',
    });

    const transfer = store.transferOwnership(owner, 1, {
      new_owner_member_id: 3,
    });
    expect(transfer).toMatchObject({
      previous_owner: { id: 1, role: 'admin' },
      new_owner: { id: 3, role: 'owner' },
    });

    const invite = store.invite(member, 1, {
      email: 'new.member@example.com',
      role: 'member',
    });
    expect(invite).toMatchObject({
      invitation: { email: 'new.member@example.com', status: 'pending' },
      email_send_status: 'not_configured',
    });
    expect(JSON.stringify(invite)).not.toContain(invitationToken);
    expect(JSON.stringify(store.listInvitations(member, 1, 1, 15))).not.toContain(
      invitationToken
    );

    const accepted = store.acceptInvitation(
      { id: 'new-member', email: 'new.member@example.com', name: 'New Member' },
      { token: invitationToken }
    );
    expect(accepted).toMatchObject({ id: 1, role: 'member' });
    expect(store.acceptInvitation(
      { id: 'new-member', email: 'new.member@example.com', name: 'New Member' },
      { token: invitationToken }
    )).toBe('invitation_invalid');
  });
});

function pageEnvelope(data: unknown[], path: string) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 15,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: `${path}?page=1`, last: `${path}?page=1`, prev: null, next: null },
  };
}

function memberView(overrides: Record<string, unknown>) {
  return {
    id: 91,
    user_id: 17,
    username: 'alex',
    nickname: 'Alex',
    role: 'member',
    joined_at: timestamp,
    updated_at: timestamp,
    ...overrides,
  };
}

function expectInvalidResponse(operation: () => unknown): void {
  try {
    operation();
    throw new Error('Expected invalid response error');
  } catch (error) {
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).errorCode).toBe(ClientErrorCode.INVALID_RESPONSE);
  }
}
