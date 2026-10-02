import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import '@/i18n';
import { OrganizationDetailPage } from '@/features/organizations/components/organization-detail-page';
import { OrganizationsPage } from '@/features/organizations/components/organizations-page';

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { 'content-type': 'application/json' },
    status,
  });
}

function page<T>(data: T[], total = data.length, lastPage = 1) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: { current_page: 1, last_page: lastPage, per_page: 20, total },
    links: { first: '', last: '', prev: null, next: null },
  };
}

function withQueryClient(children: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

const acme = {
  id: 7,
  name: 'Acme',
  slug: 'acme',
  created_by: 3,
  member_count: 2,
  created_at: '2026-09-01T00:00:00Z',
  updated_at: '2026-09-02T00:00:00Z',
};

const members = [
  {
    id: 11,
    user_id: 3,
    username: 'owner',
    nickname: 'Olive',
    email: 'owner@example.test',
    role: 'owner',
    joined_at: '2026-09-01T00:00:00Z',
  },
  {
    id: 12,
    user_id: 4,
    username: 'member',
    nickname: '',
    email: 'member@example.test',
    role: 'member',
    joined_at: '2026-09-03T00:00:00Z',
  },
];

describe('OrganizationsPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('lists organizations and opens one', async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(json(page([acme], 41, 3))));
    vi.stubGlobal('fetch', fetchMock);
    const onOpen = vi.fn();
    const onSearchChange = vi.fn();
    render(
      withQueryClient(
        <OrganizationsPage search={{ q: 'ac' }} onSearchChange={onSearchChange} onOpen={onOpen} />,
      ),
    );

    const row = (await screen.findByText('Acme')).closest('tr') as HTMLElement;
    expect(within(row).getByText('acme · #7')).toBeInTheDocument();
    expect(screen.getByText('41 organizations')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/operator/organizations?page=1&per_page=20&q=ac',
      expect.anything(),
    );

    fireEvent.click(within(row).getByRole('button', { name: 'Open Acme' }));
    expect(onOpen).toHaveBeenCalledWith(7);

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(onSearchChange).toHaveBeenCalledWith({ q: 'ac', page: 2 });
  });

  it('moves the search into the URL state and resets the page', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => Promise.resolve(json(page([])))),
    );
    const onSearchChange = vi.fn();
    render(
      withQueryClient(
        <OrganizationsPage search={{ page: 3 }} onSearchChange={onSearchChange} onOpen={vi.fn()} />,
      ),
    );

    expect(await screen.findByText(/no organizations match/i)).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/name or slug/i), { target: { value: ' glo ' } });
    fireEvent.submit(screen.getByRole('search'));
    expect(onSearchChange).toHaveBeenCalledWith({ q: 'glo', page: undefined });
  });
});

describe('OrganizationDetailPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows the organization, its members with email, and sections from other features', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockImplementation((url: string) =>
          Promise.resolve(
            url.includes('/members')
              ? json(page(members))
              : json({ code: 0, message: 'success', data: acme }),
          ),
        ),
    );
    render(
      withQueryClient(
        <OrganizationDetailPage organizationId={7} onBack={vi.fn()}>
          <p>starter section</p>
        </OrganizationDetailPage>,
      ),
    );

    expect(await screen.findByRole('heading', { level: 1, name: 'Acme' })).toBeInTheDocument();
    expect(await screen.findByText('owner · owner@example.test')).toBeInTheDocument();
    expect(screen.getByText('Olive')).toBeInTheDocument();
    expect(screen.getByText('Owner')).toBeInTheDocument();
    expect(screen.getByText('2 members')).toBeInTheDocument();
    expect(screen.getByText('starter section')).toBeInTheDocument();
  });

  it('explains an unknown organization and loads nothing beneath it', async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(
          json({ code: 404, error_code: 'ORGANIZATION.NOT_FOUND', message: 'missing' }, 404),
        ),
      );
    vi.stubGlobal('fetch', fetchMock);
    const onBack = vi.fn();
    render(
      withQueryClient(
        <OrganizationDetailPage organizationId={99} onBack={onBack}>
          <p>starter section</p>
        </OrganizationDetailPage>,
      ),
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(/does not exist/i);
    expect(screen.queryByText('starter section')).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: /all organizations/i }));
    expect(onBack).toHaveBeenCalled();
  });
});
