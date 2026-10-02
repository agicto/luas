import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import '@/i18n';
import { NotificationDeliveriesPage } from '@/features/notification-deliveries/components/notification-deliveries-page';

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

const failed = {
  id: 31,
  notification_id: 9,
  user_id: 4,
  kind: 'billing.invoice_paid',
  channel: 'email',
  status: 'failed',
  attempts: 5,
  last_failure_code: 'provider_rejected',
  available_at: '2026-09-02T00:00:00Z',
  delivered_at: null,
  created_at: '2026-09-02T00:00:00Z',
  updated_at: '2026-09-02T00:05:00Z',
};

describe('NotificationDeliveriesPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('lists delivery state with the stable failure code', async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(json(page([failed]))));
    vi.stubGlobal('fetch', fetchMock);
    render(
      withQueryClient(
        <NotificationDeliveriesPage
          search={{ status: 'failed', user_id: 4 }}
          onSearchChange={vi.fn()}
        />,
      ),
    );

    const row = (await screen.findByText('billing.invoice_paid')).closest('tr') as HTMLElement;
    expect(within(row).getByText('User #4')).toBeInTheDocument();
    expect(within(row).getByText('Email')).toBeInTheDocument();
    expect(within(row).getByText('provider_rejected')).toBeInTheDocument();
    expect(within(row).getByText(/5 attempts/)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/operator/notification-deliveries?page=1&per_page=50&status=failed&user_id=4',
      expect.anything(),
    );
  });

  it('moves filters into the URL state and ignores a non-numeric user ID', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => Promise.resolve(json(page([])))),
    );
    const onSearchChange = vi.fn();
    render(
      withQueryClient(
        <NotificationDeliveriesPage search={{ page: 2 }} onSearchChange={onSearchChange} />,
      ),
    );

    expect(await screen.findByText(/no deliveries match/i)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'failed' } });
    fireEvent.change(screen.getByLabelText('Channel'), { target: { value: 'email' } });
    fireEvent.change(screen.getByLabelText('Recipient user ID'), { target: { value: 'abc' } });
    fireEvent.submit(screen.getByRole('search'));

    expect(onSearchChange).toHaveBeenCalledWith({
      status: 'failed',
      channel: 'email',
      user_id: undefined,
      page: undefined,
    });
  });
});
