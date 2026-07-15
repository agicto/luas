import { fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { OrganizationDirectory } from '@/features/organization/components/organization-directory';
import { OrganizationOverview } from '@/features/organization/components/organization-overview';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import { messages } from '@/i18n/modules';

const state = vi.hoisted(() => ({
  organizations: {
    data: undefined as unknown,
    error: null as unknown,
    isPending: false,
    refetch: vi.fn(),
  },
  context: {
    data: undefined as unknown,
    error: null as unknown,
    isPending: false,
    refetch: vi.fn(),
  },
  create: {
    error: null as unknown,
    isPending: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
  update: {
    error: null as unknown,
    isPending: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
}));

vi.mock('@/features/organization/hooks/use-organizations', () => ({
  useOrganizations: () => state.organizations,
  useOrganizationContext: () => state.context,
  useCreateOrganization: () => state.create,
  useUpdateOrganization: () => state.update,
}));

function withMessages(children: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}

function renderWithMessages(children: React.ReactNode) {
  return render(withMessages(children));
}

describe('organization browser workflow', () => {
  beforeEach(() => {
    state.organizations.data = undefined;
    state.organizations.error = null;
    state.organizations.isPending = false;
    state.organizations.refetch.mockReset();
    state.context.data = undefined;
    state.context.error = null;
    state.context.isPending = false;
    state.context.refetch.mockReset();
    state.create.error = null;
    state.create.isPending = false;
    state.create.mutate.mockReset();
    state.create.reset.mockReset();
    state.update.error = null;
    state.update.isPending = false;
    state.update.mutate.mockReset();
    state.update.reset.mockReset();
  });

  it('renders organization identity, immutable slug, and role from the validated page', () => {
    state.organizations.data = {
      items: [
        {
          id: 42,
          name: 'Acme Europe',
          slug: 'acme-europe',
          role: 'owner',
          created_at: '2026-07-15T10:00:00Z',
          updated_at: '2026-07-15T10:00:00Z',
        },
      ],
      meta: { total: 1 },
    };

    renderWithMessages(<OrganizationDirectory />);

    expect(screen.getByRole('heading', { name: 'Organizations' })).toBeInTheDocument();
    expect(screen.getByText('Acme Europe')).toBeInTheDocument();
    expect(screen.getAllByText('acme-europe')).toHaveLength(2);
    expect(screen.getByText('Owner')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open Acme Europe' })).toHaveAttribute(
      'href',
      '/console/organizations/42'
    );
  });

  it('uses local copy for malformed successful responses', () => {
    state.organizations.error = new ApiError(
      'raw parser detail',
      ClientErrorCode.INVALID_RESPONSE
    );

    renderWithMessages(<OrganizationDirectory />);

    expect(screen.getByRole('alert')).toHaveTextContent(
      'The organization service returned data this client cannot recognize.'
    );
    expect(screen.queryByText('raw parser detail')).not.toBeInTheDocument();
  });

  it('keeps a member organization profile read-only', () => {
    state.context.data = {
      organization_id: 42,
      organization_name: 'Acme Europe',
      organization_slug: 'acme-europe',
      membership_id: 91,
      user_id: 17,
      role: 'member',
    };

    renderWithMessages(<OrganizationOverview organizationId={42} />);

    expect(screen.getByLabelText('Organization name')).toBeDisabled();
    expect(screen.getByLabelText('Organization slug')).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    expect(screen.getByText('Context verified')).toBeInTheDocument();
  });

  it('isolates editable state when switching between cached organization contexts', () => {
    state.context.data = {
      organization_id: 42,
      organization_name: 'Acme Europe',
      organization_slug: 'acme-europe',
      membership_id: 91,
      user_id: 17,
      role: 'owner',
    };
    const view = renderWithMessages(<OrganizationOverview organizationId={42} />);

    expect(screen.getByLabelText('Organization name')).toHaveValue('Acme Europe');

    state.context.data = {
      organization_id: 43,
      organization_name: 'Acme Americas',
      organization_slug: 'acme-americas',
      membership_id: 92,
      user_id: 17,
      role: 'admin',
    };
    view.rerender(withMessages(<OrganizationOverview organizationId={43} />));

    expect(screen.getByLabelText('Organization name')).toHaveValue('Acme Americas');
  });

  it('associates server field ownership with the organization name control', () => {
    state.context.data = {
      organization_id: 42,
      organization_name: 'Acme Europe',
      organization_slug: 'acme-europe',
      membership_id: 91,
      user_id: 17,
      role: 'owner',
    };
    state.update.error = new ApiError(
      'raw backend validation detail',
      ApiErrorCode.COMMON_VALIDATION_FAILED,
      422,
      'request-1',
      { name: ['raw field detail'] }
    );

    renderWithMessages(<OrganizationOverview organizationId={42} />);

    const name = screen.getByLabelText('Organization name');
    expect(name).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByText('Enter an organization name between 2 and 100 characters')).toBeInTheDocument();
    expect(screen.queryByText('raw backend validation detail')).not.toBeInTheDocument();
    fireEvent.change(name, { target: { value: 'Acme Global' } });
    expect(state.update.reset).toHaveBeenCalledTimes(1);
  });
});
