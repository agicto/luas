import { describe, expect, it } from 'vitest';

import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import { MockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import { createOrganizationSchema } from '@/features/organization/schemas';
import {
  parseOrganizationContextResponse,
  parseOrganizationPageResponse,
  parseOrganizationResponse,
} from '@/features/organization/services/organization-service';

const timestamp = '2026-07-15T10:00:00.000Z';
const organization = {
  id: 42,
  name: 'Acme Europe',
  slug: 'acme-europe',
  role: 'owner',
  created_at: timestamp,
  updated_at: timestamp,
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
});

describe('MockOrganizationStore', () => {
  it('supports the list, create, resolve, and rename browser workflow', () => {
    const store = new MockOrganizationStore({
      now: () => new Date(timestamp),
      randomUUID: () => '01234567-89ab-cdef-0123-456789abcdef',
    });

    const created = store.create({ name: 'Research Lab' });
    expect(created).toMatchObject({
      id: 2,
      name: 'Research Lab',
      slug: 'org-0123456789abcdef01234567',
      role: 'owner',
    });
    expect(store.list(1, 15).meta.total).toBe(2);
    expect(store.resolveContext(2)).toMatchObject({
      organization_id: 2,
      membership_id: 2,
      role: 'owner',
    });
    expect(store.update(2, { name: 'Research Europe' })).toMatchObject({
      name: 'Research Europe',
      slug: 'org-0123456789abcdef01234567',
    });
    expect(store.list(1, 7).links).toMatchObject({
      first: '/api/organizations?page=1&per_page=7',
      last: '/api/organizations?page=1&per_page=7',
    });
  });

  it('preserves global mock slug uniqueness', () => {
    const store = new MockOrganizationStore();
    expect(store.create({ name: 'First', slug: 'shared-slug' })).not.toBe('slug_conflict');
    expect(store.create({ name: 'Second', slug: 'shared-slug' })).toBe('slug_conflict');
  });
});

function expectInvalidResponse(operation: () => unknown): void {
  try {
    operation();
    throw new Error('Expected invalid response error');
  } catch (error) {
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).errorCode).toBe(ClientErrorCode.INVALID_RESPONSE);
  }
}
