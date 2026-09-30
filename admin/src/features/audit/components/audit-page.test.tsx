import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import '@/i18n';
import { AuditPage } from '@/features/audit/components/audit-page';

function renderWith(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response));
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <AuditPage search={{}} onSearchChange={vi.fn()} />
    </QueryClientProvider>,
  );
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { 'content-type': 'application/json' },
    status,
  });
}

describe('AuditPage', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('shows the actor, action, and request of each entry', async () => {
    renderWith(
      json({
        code: 0,
        message: 'success',
        data: [
          {
            id: 1,
            user_id: 1,
            actor_type: 'user',
            action: 'disable',
            resource: 'users',
            method: 'POST',
            path: '/v1/operator/users/:id/disable',
            status_code: 200,
            request_id: 'req-1',
            created_at: '2026-09-30T04:40:00Z',
          },
        ],
        meta: { current_page: 1, last_page: 1, per_page: 50, total: 1 },
      }),
    );

    expect(await screen.findByText('User #1')).toBeInTheDocument();
    expect(screen.getByText('POST /v1/operator/users/:id/disable')).toBeInTheDocument();
    expect(screen.getByText('req-1')).toBeInTheDocument();
  });

  it('explains an invalid date range', async () => {
    renderWith(json({ code: 400, error_code: 'COMMON.INVALID_INPUT', message: 'bad' }, 400));
    expect(await screen.findByRole('alert')).toHaveTextContent(/92 days/);
  });
});
