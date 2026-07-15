import 'server-only';

import type {
  CreateOrganizationInput,
  Organization,
  OrganizationContext,
  OrganizationPage,
  UpdateOrganizationInput,
} from '@/features/organization/types';

interface MockOrganizationStoreDependencies {
  now: () => Date;
  randomUUID: () => string;
}

export class MockOrganizationStore {
  private organizations: Organization[];
  private nextOrganizationId = 2;
  private readonly dependencies: MockOrganizationStoreDependencies;

  constructor(dependencies: Partial<MockOrganizationStoreDependencies> = {}) {
    this.dependencies = {
      now: dependencies.now ?? (() => new Date()),
      randomUUID: dependencies.randomUUID ?? (() => crypto.randomUUID()),
    };
    const timestamp = this.dependencies.now().toISOString();
    this.organizations = [
      {
        id: 1,
        name: 'Luas Demo',
        slug: 'luas-demo',
        role: 'owner',
        created_at: timestamp,
        updated_at: timestamp,
      },
    ];
  }

  list(page: number, perPage: number): OrganizationPage {
    const start = (page - 1) * perPage;
    const total = this.organizations.length;
    const lastPage = Math.max(1, Math.ceil(total / perPage));
    const items = this.organizations.slice(start, start + perPage).map(cloneOrganization);
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
        first: paginationLink(1, perPage),
        last: paginationLink(lastPage, perPage),
        prev: page > 1 ? paginationLink(page - 1, perPage) : null,
        next: page < lastPage ? paginationLink(page + 1, perPage) : null,
      },
    };
  }

  get(organizationId: number): Organization | null {
    const organization = this.organizations.find(({ id }) => id === organizationId);
    return organization ? cloneOrganization(organization) : null;
  }

  create(input: CreateOrganizationInput): Organization | 'slug_conflict' {
    const slug = input.slug ?? this.generatedSlug();
    if (this.organizations.some((organization) => organization.slug === slug)) {
      return 'slug_conflict';
    }

    const timestamp = this.dependencies.now().toISOString();
    const organization: Organization = {
      id: this.nextOrganizationId++,
      name: input.name,
      slug,
      role: 'owner',
      created_at: timestamp,
      updated_at: timestamp,
    };
    this.organizations = [organization, ...this.organizations];
    return cloneOrganization(organization);
  }

  update(
    organizationId: number,
    input: UpdateOrganizationInput
  ): Organization | null {
    const index = this.organizations.findIndex(({ id }) => id === organizationId);
    if (index < 0) {
      return null;
    }

    const updated: Organization = {
      ...this.organizations[index],
      name: input.name,
      updated_at: this.dependencies.now().toISOString(),
    };
    this.organizations = this.organizations.map((organization, currentIndex) =>
      currentIndex === index ? updated : organization
    );
    return cloneOrganization(updated);
  }

  resolveContext(organizationId: number): OrganizationContext | null {
    const organization = this.get(organizationId);
    if (!organization) {
      return null;
    }

    return {
      organization_id: organization.id,
      organization_name: organization.name,
      organization_slug: organization.slug,
      membership_id: organization.id,
      user_id: 1,
      role: organization.role,
    };
  }

  private generatedSlug(): string {
    const suffix = this.dependencies.randomUUID().replace(/[^a-z0-9]/gi, '').toLowerCase();
    return `org-${suffix.slice(0, 24)}`;
  }
}

export const mockOrganizationStore = new MockOrganizationStore();

function cloneOrganization(organization: Organization): Organization {
  return { ...organization };
}

function paginationLink(page: number, perPage: number): string {
  return `/api/organizations?page=${page}&per_page=${perPage}`;
}
