import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import '@/i18n';
import { WebhookDiagnostics } from '@/features/webhooks/components/webhook-diagnostics';

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

const endpoint = {
  id: 5,
  organization_id: 7,
  name: 'Receiver',
  url: 'https://hooks.example.test/luas',
  event_types: ['webhook.test'],
  status: 'disabled',
  disabled_reason: 'consecutive_failures',
  consecutive_failures: 8,
  version: 3,
  secret_hint: 'abcd',
  secret_version: 1,
  previous_secret_expiry: null,
  created_at: '2026-09-01T00:00:00Z',
  updated_at: '2026-09-02T00:00:00Z',
};

function delivery(id: number, status: string) {
  return {
    id,
    endpoint_id: 5,
    message_id: `msg_${id}`,
    event_type: 'webhook.test',
    status,
    attempt_count: 3,
    replay_count: 0,
    http_status: status === 'failed' ? 500 : null,
    failure_code: status === 'failed' ? 'http_status' : '',
    response_truncated: false,
    available_at: '2026-09-02T00:00:00Z',
    delivered_at: null,
    created_at: '2026-09-02T00:00:00Z',
    updated_at: '2026-09-02T00:05:00Z',
  };
}

const attempt = {
  id: 1,
  delivery_id: 21,
  number: 3,
  outcome: 'failed',
  http_status: 500,
  failure_code: 'http_status',
  duration_ms: 120,
  response_truncated: false,
  started_at: '2026-09-02T00:04:59Z',
  completed_at: '2026-09-02T00:05:00Z',
};

function stubApi(replayResponse: () => Response) {
  const fetchMock = vi.fn().mockImplementation((url: string, init: RequestInit) => {
    if (init.method === 'POST') {
      return Promise.resolve(replayResponse());
    }
    if (url.includes('/attempts')) {
      return Promise.resolve(json(page([attempt])));
    }
    if (url.includes('/webhook-endpoints')) {
      return Promise.resolve(json(page([endpoint])));
    }
    return Promise.resolve(json(page([delivery(21, 'failed'), delivery(22, 'pending')])));
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

describe('WebhookDiagnostics', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows endpoints without secrets and offers replay only for finished deliveries', async () => {
    stubApi(() => json({}));
    render(withQueryClient(<WebhookDiagnostics organizationId={7} />));

    expect(await screen.findByText('https://hooks.example.test/luas')).toBeInTheDocument();
    expect(screen.getByText('consecutive_failures')).toBeInTheDocument();
    expect(screen.getByText('8 consecutive failures')).toBeInTheDocument();
    expect(screen.queryByText(/abcd/)).not.toBeInTheDocument();

    const failedRow = (await screen.findByText(/msg_21/)).closest('tr') as HTMLElement;
    const pendingRow = screen.getByText(/msg_22/).closest('tr') as HTMLElement;
    expect(within(failedRow).getByRole('button', { name: 'Replay' })).toBeInTheDocument();
    expect(within(pendingRow).queryByRole('button', { name: 'Replay' })).not.toBeInTheDocument();
    expect(within(failedRow).getByText('500 · http_status')).toBeInTheDocument();
  });

  it('replays a delivery only after confirmation', async () => {
    const fetchMock = stubApi(() =>
      json({ code: 0, message: 'success', data: { ...delivery(21, 'pending'), replay_count: 1 } }),
    );
    render(withQueryClient(<WebhookDiagnostics organizationId={7} />));

    fireEvent.click(await screen.findByRole('button', { name: 'Replay' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/msg_21/)).toBeInTheDocument();
    expect(fetchMock.mock.calls.some(([, init]) => (init as RequestInit).method === 'POST')).toBe(
      false,
    );

    fireEvent.click(within(dialog).getByRole('button', { name: 'Replay' }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/v1/operator/organizations/7/webhook-deliveries/21/replay',
        expect.objectContaining({ method: 'POST' }),
      ),
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('explains a rejected replay inside the dialog', async () => {
    stubApi(() =>
      json({ code: 409, error_code: 'WEBHOOK.REPLAY_NOT_ALLOWED', message: 'no' }, 409),
    );
    render(withQueryClient(<WebhookDiagnostics organizationId={7} />));

    fireEvent.click(await screen.findByRole('button', { name: 'Replay' }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Replay' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/cannot be replayed/i);
  });

  it('lists attempts and filters deliveries by status', async () => {
    const fetchMock = stubApi(() => json({}));
    render(withQueryClient(<WebhookDiagnostics organizationId={7} />));

    const failedRow = (await screen.findByText(/msg_21/)).closest('tr') as HTMLElement;
    fireEvent.click(within(failedRow).getByRole('button', { name: 'Attempts' }));
    const dialog = await screen.findByRole('dialog');
    expect(await within(dialog).findByText('Attempt 3')).toBeInTheDocument();
    expect(within(dialog).getByText('failed · 500 · http_status')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/operator/organizations/7/webhook-deliveries/21/attempts?per_page=50',
      expect.anything(),
    );
    fireEvent.click(within(dialog).getByRole('button', { name: 'Close' }));

    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'failed' } });
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/v1/operator/organizations/7/webhook-deliveries?page=1&per_page=20&status=failed',
        expect.anything(),
      ),
    );
  });
});
