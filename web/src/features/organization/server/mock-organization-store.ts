import 'server-only';

import type { AuthUser } from '@/features/auth/types';
import type {
  AcceptOrganizationInvitationInput,
  CreateOrganizationInput,
  CreateOrganizationInvitationInput,
  Organization,
  OrganizationContext,
  OrganizationInvitation,
  OrganizationInvitationCreateResult,
  OrganizationInvitationPage,
  OrganizationInvitationStatus,
  OrganizationMember,
  OrganizationMemberPage,
  OrganizationOwnershipTransfer,
  OrganizationPage,
  OrganizationRole,
  TransferOrganizationOwnershipInput,
  UpdateOrganizationInput,
  UpdateOrganizationMemberInput,
} from '@/features/organization/types';

const INVITATION_TTL_MS = 7 * 24 * 60 * 60 * 1000;

export type MockOrganizationStoreError =
  | 'invitation_already_pending'
  | 'invitation_email_mismatch'
  | 'invitation_expired'
  | 'invitation_invalid'
  | 'invitation_not_found'
  | 'member_already_exists'
  | 'member_not_found'
  | 'organization_not_found'
  | 'ownership_transfer_required'
  | 'ownership_transfer_target_invalid'
  | 'permission_denied'
  | 'slug_conflict';

interface MockOrganizationStoreDependencies {
  now: () => Date;
  randomUUID: () => string;
  invitationToken: () => string;
}

interface MockOrganizationRecord {
  id: number;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
}

interface MockUserRecord {
  key: string;
  id: number;
  email: string;
  username: string;
  nickname: string;
  avatar?: string;
}

interface MockMembershipRecord {
  id: number;
  organizationId: number;
  userKey: string;
  role: OrganizationRole;
  joinedAt: string;
  updatedAt: string;
}

interface MockInvitationRecord extends OrganizationInvitation {
  token: string;
}

export class MockOrganizationStore {
  private organizations: MockOrganizationRecord[];
  private users: MockUserRecord[];
  private memberships: MockMembershipRecord[];
  private invitations: MockInvitationRecord[] = [];
  private nextOrganizationId = 2;
  private nextUserId = 4;
  private nextMembershipId = 4;
  private nextInvitationId = 1;
  private readonly dependencies: MockOrganizationStoreDependencies;

  constructor(dependencies: Partial<MockOrganizationStoreDependencies> = {}) {
    const randomUUID = dependencies.randomUUID ?? (() => crypto.randomUUID());
    this.dependencies = {
      now: dependencies.now ?? (() => new Date()),
      randomUUID,
      invitationToken:
        dependencies.invitationToken ??
        (() => `oinv_mock.${randomUUID().replaceAll('-', '').toLowerCase()}`),
    };

    const timestamp = this.dependencies.now().toISOString();
    this.organizations = [
      {
        id: 1,
        name: 'Luas Demo',
        slug: 'luas-demo',
        createdAt: timestamp,
        updatedAt: timestamp,
      },
    ];
    this.users = [
      mockUser(1, 'demo-admin', 'admin@example.com', 'admin', 'Admin User'),
      mockUser(2, 'demo-operator', 'operator@example.com', 'morgan', 'Morgan Lee'),
      mockUser(3, 'demo-member', 'member@example.com', 'riley', 'Riley Chen'),
    ];
    this.memberships = [
      mockMembership(1, 1, 'demo-admin', 'owner', timestamp),
      mockMembership(2, 1, 'demo-operator', 'admin', timestamp),
      mockMembership(3, 1, 'demo-member', 'member', timestamp),
    ];
  }

  list(actor: AuthUser, page: number, perPage: number): OrganizationPage {
    const memberships = this.memberships.filter(
      (membership) => membership.userKey === actor.id
    );
    const items = memberships.flatMap((membership) => {
      const organization = this.organizationRecord(membership.organizationId);
      return organization ? [this.organizationView(organization, membership.role)] : [];
    });
    return paginated(items, page, perPage, '/api/organizations');
  }

  get(actor: AuthUser, organizationId: number): Organization | null {
    const membership = this.membershipForActor(actor, organizationId);
    const organization = this.organizationRecord(organizationId);
    return membership && organization
      ? this.organizationView(organization, membership.role)
      : null;
  }

  create(
    actor: AuthUser,
    input: CreateOrganizationInput
  ): Organization | 'slug_conflict' {
    const slug = input.slug ?? this.generatedSlug();
    if (this.organizations.some((organization) => organization.slug === slug)) {
      return 'slug_conflict';
    }

    this.ensureUser(actor);
    const timestamp = this.dependencies.now().toISOString();
    const organization: MockOrganizationRecord = {
      id: this.nextOrganizationId++,
      name: input.name,
      slug,
      createdAt: timestamp,
      updatedAt: timestamp,
    };
    this.organizations = [organization, ...this.organizations];
    this.memberships = [
      ...this.memberships,
      mockMembership(
        this.nextMembershipId++,
        organization.id,
        actor.id,
        'owner',
        timestamp
      ),
    ];
    return this.organizationView(organization, 'owner');
  }

  update(
    actor: AuthUser,
    organizationId: number,
    input: UpdateOrganizationInput
  ): Organization | MockOrganizationStoreError {
    const membership = this.membershipForActor(actor, organizationId);
    const organization = this.organizationRecord(organizationId);
    if (!membership || !organization) return 'organization_not_found';
    if (!canManageOrganization(membership.role)) return 'permission_denied';

    organization.name = input.name;
    organization.updatedAt = this.dependencies.now().toISOString();
    return this.organizationView(organization, membership.role);
  }

  resolveContext(
    actor: AuthUser,
    organizationId: number
  ): OrganizationContext | null {
    const membership = this.membershipForActor(actor, organizationId);
    const organization = this.organizationRecord(organizationId);
    const user = this.userRecord(actor.id);
    if (!membership || !organization || !user) return null;

    return {
      organization_id: organization.id,
      organization_name: organization.name,
      organization_slug: organization.slug,
      membership_id: membership.id,
      user_id: user.id,
      role: membership.role,
    };
  }

  listMembers(
    actor: AuthUser,
    organizationId: number,
    page: number,
    perPage: number
  ): OrganizationMemberPage | 'organization_not_found' {
    if (!this.membershipForActor(actor, organizationId)) {
      return 'organization_not_found';
    }
    const items = this.memberships
      .filter((membership) => membership.organizationId === organizationId)
      .flatMap((membership) => {
        const user = this.userRecord(membership.userKey);
        return user ? [memberView(membership, user)] : [];
      });
    return paginated(
      items,
      page,
      perPage,
      `/api/organizations/${organizationId}/members`
    );
  }

  changeMemberRole(
    actor: AuthUser,
    organizationId: number,
    memberId: number,
    input: UpdateOrganizationMemberInput
  ): OrganizationMember | MockOrganizationStoreError {
    const actingMembership = this.membershipForActor(actor, organizationId);
    if (!actingMembership) return 'organization_not_found';
    if (actingMembership.role !== 'owner') return 'permission_denied';

    const target = this.membershipRecord(organizationId, memberId);
    if (!target) return 'member_not_found';
    if (target.role === 'owner') return 'ownership_transfer_required';
    const user = this.userRecord(target.userKey);
    if (!user) return 'member_not_found';

    target.role = input.role;
    target.updatedAt = this.dependencies.now().toISOString();
    return memberView(target, user);
  }

  removeMember(
    actor: AuthUser,
    organizationId: number,
    memberId: number
  ): true | MockOrganizationStoreError {
    const actingMembership = this.membershipForActor(actor, organizationId);
    if (!actingMembership) return 'organization_not_found';
    const target = this.membershipRecord(organizationId, memberId);
    if (!target) return 'member_not_found';

    if (target.id === actingMembership.id) {
      if (actingMembership.role === 'owner') {
        return 'ownership_transfer_required';
      }
    } else if (!canRemoveMember(actingMembership.role, target.role)) {
      return 'permission_denied';
    }

    this.memberships = this.memberships.filter(
      (membership) => membership.id !== target.id
    );
    return true;
  }

  transferOwnership(
    actor: AuthUser,
    organizationId: number,
    input: TransferOrganizationOwnershipInput
  ): OrganizationOwnershipTransfer | MockOrganizationStoreError {
    const previousOwner = this.membershipForActor(actor, organizationId);
    if (!previousOwner) return 'organization_not_found';
    if (previousOwner.role !== 'owner') return 'permission_denied';

    const newOwner = this.membershipRecord(
      organizationId,
      input.new_owner_member_id
    );
    if (!newOwner || newOwner.id === previousOwner.id || newOwner.role === 'owner') {
      return 'ownership_transfer_target_invalid';
    }
    const previousOwnerUser = this.userRecord(previousOwner.userKey);
    const newOwnerUser = this.userRecord(newOwner.userKey);
    if (!previousOwnerUser || !newOwnerUser) {
      return 'ownership_transfer_target_invalid';
    }

    const timestamp = this.dependencies.now().toISOString();
    previousOwner.role = 'admin';
    previousOwner.updatedAt = timestamp;
    newOwner.role = 'owner';
    newOwner.updatedAt = timestamp;
    return {
      previous_owner: memberView(previousOwner, previousOwnerUser),
      new_owner: memberView(newOwner, newOwnerUser),
    };
  }

  listInvitations(
    actor: AuthUser,
    organizationId: number,
    page: number,
    perPage: number
  ): OrganizationInvitationPage | MockOrganizationStoreError {
    const managerError = this.managerError(actor, organizationId);
    if (managerError) return managerError;

    const items = this.invitations
      .filter((invitation) => invitation.organization_id === organizationId)
      .map((invitation) => publicInvitation(invitation, this.dependencies.now()));
    return paginated(
      items,
      page,
      perPage,
      `/api/organizations/${organizationId}/invitations`
    );
  }

  invite(
    actor: AuthUser,
    organizationId: number,
    input: CreateOrganizationInvitationInput
  ): OrganizationInvitationCreateResult | MockOrganizationStoreError {
    const managerError = this.managerError(actor, organizationId);
    if (managerError) return managerError;
    const memberExists = this.memberships.some((membership) => {
      if (membership.organizationId !== organizationId) return false;
      return this.userRecord(membership.userKey)?.email === input.email;
    });
    if (memberExists) return 'member_already_exists';

    const now = this.dependencies.now();
    const alreadyPending = this.invitations.some(
      (invitation) =>
        invitation.organization_id === organizationId &&
        invitation.email === input.email &&
        invitationStatus(invitation, now) === 'pending'
    );
    if (alreadyPending) return 'invitation_already_pending';

    const timestamp = now.toISOString();
    const invitation: MockInvitationRecord = {
      id: this.nextInvitationId++,
      organization_id: organizationId,
      email: input.email,
      role: input.role,
      status: 'pending',
      expires_at: new Date(now.getTime() + INVITATION_TTL_MS).toISOString(),
      created_at: timestamp,
      updated_at: timestamp,
      token: this.dependencies.invitationToken(),
    };
    this.invitations = [invitation, ...this.invitations];
    return {
      invitation: publicInvitation(invitation, now),
      email_send_status: 'not_configured',
    };
  }

  revokeInvitation(
    actor: AuthUser,
    organizationId: number,
    invitationId: number
  ): true | MockOrganizationStoreError {
    const managerError = this.managerError(actor, organizationId);
    if (managerError) return managerError;
    const invitation = this.invitations.find(
      (candidate) =>
        candidate.organization_id === organizationId &&
        candidate.id === invitationId
    );
    if (!invitation || invitationStatus(invitation, this.dependencies.now()) !== 'pending') {
      return 'invitation_not_found';
    }
    invitation.status = 'revoked';
    invitation.updated_at = this.dependencies.now().toISOString();
    return true;
  }

  acceptInvitation(
    actor: AuthUser,
    input: AcceptOrganizationInvitationInput
  ): Organization | MockOrganizationStoreError {
    const invitation = this.invitations.find(
      (candidate) => candidate.token === input.token
    );
    if (!invitation) return 'invitation_invalid';
    const status = invitationStatus(invitation, this.dependencies.now());
    if (status === 'expired') return 'invitation_expired';
    if (status !== 'pending') return 'invitation_invalid';

    const user = this.ensureUser(actor);
    if (user.email !== invitation.email) return 'invitation_email_mismatch';
    if (this.membershipForActor(actor, invitation.organization_id)) {
      return 'member_already_exists';
    }
    const organization = this.organizationRecord(invitation.organization_id);
    if (!organization) return 'invitation_invalid';

    const timestamp = this.dependencies.now().toISOString();
    invitation.status = 'accepted';
    invitation.updated_at = timestamp;
    this.memberships = [
      ...this.memberships,
      mockMembership(
        this.nextMembershipId++,
        invitation.organization_id,
        actor.id,
        invitation.role,
        timestamp
      ),
    ];
    return this.organizationView(organization, invitation.role);
  }

  private managerError(
    actor: AuthUser,
    organizationId: number
  ): 'organization_not_found' | 'permission_denied' | null {
    const membership = this.membershipForActor(actor, organizationId);
    if (!membership) return 'organization_not_found';
    return canManageOrganization(membership.role) ? null : 'permission_denied';
  }

  private ensureUser(actor: AuthUser): MockUserRecord {
    const existing = this.userRecord(actor.id);
    if (existing) return existing;
    const user = mockUser(
      this.nextUserId++,
      actor.id,
      actor.email.toLowerCase(),
      usernameFromActor(actor),
      actor.name
    );
    this.users = [...this.users, user];
    return user;
  }

  private userRecord(key: string): MockUserRecord | undefined {
    return this.users.find((user) => user.key === key);
  }

  private organizationRecord(id: number): MockOrganizationRecord | undefined {
    return this.organizations.find((organization) => organization.id === id);
  }

  private membershipForActor(
    actor: AuthUser,
    organizationId: number
  ): MockMembershipRecord | undefined {
    return this.memberships.find(
      (membership) =>
        membership.organizationId === organizationId &&
        membership.userKey === actor.id
    );
  }

  private membershipRecord(
    organizationId: number,
    memberId: number
  ): MockMembershipRecord | undefined {
    return this.memberships.find(
      (membership) =>
        membership.organizationId === organizationId && membership.id === memberId
    );
  }

  private organizationView(
    organization: MockOrganizationRecord,
    role: OrganizationRole
  ): Organization {
    return {
      id: organization.id,
      name: organization.name,
      slug: organization.slug,
      role,
      created_at: organization.createdAt,
      updated_at: organization.updatedAt,
    };
  }

  private generatedSlug(): string {
    const suffix = this.dependencies
      .randomUUID()
      .replace(/[^a-z0-9]/gi, '')
      .toLowerCase();
    return `org-${suffix.slice(0, 24)}`;
  }
}

export const mockOrganizationStore = new MockOrganizationStore();

function mockUser(
  id: number,
  key: string,
  email: string,
  username: string,
  nickname: string
): MockUserRecord {
  return { id, key, email, username, nickname };
}

function mockMembership(
  id: number,
  organizationId: number,
  userKey: string,
  role: OrganizationRole,
  timestamp: string
): MockMembershipRecord {
  return {
    id,
    organizationId,
    userKey,
    role,
    joinedAt: timestamp,
    updatedAt: timestamp,
  };
}

function memberView(
  membership: MockMembershipRecord,
  user: MockUserRecord
): OrganizationMember {
  return {
    id: membership.id,
    user_id: user.id,
    username: user.username,
    nickname: user.nickname,
    ...(user.avatar ? { avatar: user.avatar } : {}),
    role: membership.role,
    joined_at: membership.joinedAt,
    updated_at: membership.updatedAt,
  };
}

function publicInvitation(
  invitation: MockInvitationRecord,
  now: Date
): OrganizationInvitation {
  return {
    id: invitation.id,
    organization_id: invitation.organization_id,
    email: invitation.email,
    role: invitation.role,
    status: invitationStatus(invitation, now),
    expires_at: invitation.expires_at,
    created_at: invitation.created_at,
    updated_at: invitation.updated_at,
  };
}

function invitationStatus(
  invitation: MockInvitationRecord,
  now: Date
): OrganizationInvitationStatus {
  if (invitation.status === 'accepted' || invitation.status === 'revoked') {
    return invitation.status;
  }
  return now.getTime() >= Date.parse(invitation.expires_at)
    ? 'expired'
    : 'pending';
}

function canManageOrganization(role: OrganizationRole): boolean {
  return role === 'owner' || role === 'admin';
}

function canRemoveMember(
  actorRole: OrganizationRole,
  targetRole: OrganizationRole
): boolean {
  if (actorRole === 'owner') return targetRole !== 'owner';
  return actorRole === 'admin' && targetRole === 'member';
}

function usernameFromActor(actor: AuthUser): string {
  const local = actor.email.split('@', 1)[0]?.replace(/[^a-z0-9_-]/gi, '');
  return local || `user-${actor.id.slice(0, 12)}`;
}

function paginated<T>(
  values: T[],
  page: number,
  perPage: number,
  path: string
): { items: T[]; meta: OrganizationPage['meta']; links: OrganizationPage['links'] } {
  const start = (page - 1) * perPage;
  const total = values.length;
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const items = values.slice(start, start + perPage);
  const from = items.length === 0 ? 0 : start + 1;
  const to = items.length === 0 ? 0 : start + items.length;

  return {
    items,
    meta: {
      current_page: page,
      per_page: perPage,
      total,
      last_page: lastPage,
      from,
      to,
    },
    links: {
      first: paginationLink(path, 1, perPage),
      last: paginationLink(path, lastPage, perPage),
      prev: page > 1 ? paginationLink(path, page - 1, perPage) : null,
      next: page < lastPage ? paginationLink(path, page + 1, perPage) : null,
    },
  };
}

function paginationLink(path: string, page: number, perPage: number): string {
  return `${path}?page=${page}&per_page=${perPage}`;
}
