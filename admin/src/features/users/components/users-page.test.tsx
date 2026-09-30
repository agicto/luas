import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import '@/i18n';
import { UsersPage } from '@/features/users/components/users-page';

const page = {
  code: 0,
  message: 'success',
  data: [
    {
      id: 2,
      username: 'member',
      email: 'member@example.test',
      nickname: 'Member',
      status: 'active',
      is_operator: false,
      created_at: '2026-09-01T00:00:00Z',
      last_login: null,
    },
    {
      id: 1,
      username: 'ops',
      email: 'ops@example.test',
      nickname: 'Ops',
      status: 'active',
      is_operator: true,
      created_at: '2026-09-01T00:00:00Z',
      last_login: '2026-09-30T08:00:00Z',
    },
  ],
  meta: { current_page: 1, last_page: 1, per_page: 20, total: 2 },
  links: { first: '', last: '', prev: null, next: null },
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { 'content-type': 'application/json' },
    status,
  });
}

function renderPage(onSearchChange = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <UsersPage search={{}} onSearchChange={onSearchChange} />
    </QueryClientProvider>,
  );
  return onSearchChange;
}

describe('UsersPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('lists accounts and offers no actions on operator accounts', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => Promise.resolve(json(page))),
    );
    renderPage();

    const memberRow = (await screen.findByText('member · member@example.test')).closest('tr');
    const operatorRow = screen.getByText('ops · ops@example.test').closest('tr');
    expect(
      within(memberRow as HTMLElement).getByRole('button', { name: 'Disable' }),
    ).toBeInTheDocument();
    expect(within(operatorRow as HTMLElement).queryByRole('button')).not.toBeInTheDocument();
    expect(
      within(operatorRow as HTMLElement).getByText(/managed from the cli/i),
    ).toBeInTheDocument();
  });

  it('disables an account only after confirmation', async () => {
    const fetchMock = vi.fn().mockImplementation((_url: string, init: RequestInit) => {
      if (init.method === 'POST') {
        return Promise.resolve(
          json({ code: 0, message: 'success', data: { ...page.data[0], status: 'disabled' } }),
        );
      }
      return Promise.resolve(json(page));
    });
    vi.stubGlobal('fetch', fetchMock);
    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: 'Disable' }));
    const dialog = await screen.findByRole('dialog');
    expect(fetchMock.mock.calls.some(([, init]) => (init as RequestInit).method === 'POST')).toBe(
      false,
    );

    fireEvent.click(within(dialog).getByRole('button', { name: 'Disable' }));

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/v1/operator/users/2/disable',
        expect.objectContaining({ method: 'POST' }),
      ),
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('explains a protected-target rejection inside the dialog', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockImplementation((_url: string, init: RequestInit) =>
          Promise.resolve(
            init.method === 'POST'
              ? json(
                  { code: 409, error_code: 'OPERATOR.TARGET_PROTECTED', message: 'protected' },
                  409,
                )
              : json(page),
          ),
        ),
    );
    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: /end sessions/i }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: /end sessions/i }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/server command line/i);
  });

  it('moves the search into the URL state', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => Promise.resolve(json(page))),
    );
    const onSearchChange = renderPage();

    fireEvent.change(await screen.findByPlaceholderText(/username or email/i), {
      target: { value: ' mem ' },
    });
    fireEvent.submit(screen.getByRole('search'));

    expect(onSearchChange).toHaveBeenCalledWith({ q: 'mem', page: undefined });
  });
});
