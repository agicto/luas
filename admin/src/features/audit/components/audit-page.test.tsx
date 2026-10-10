import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import '@/i18n';
import { AuditPage } from '@/features/audit/components/audit-page';
import type { AuditSearch } from '@/features/audit/types';

function renderWith(response: Response, search: AuditSearch = {}, onSearchChange = vi.fn()) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response));
  const view = render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <AuditPage search={search} onSearchChange={onSearchChange} />
    </QueryClientProvider>,
  );
  return { ...view, onSearchChange };
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
        meta: { per_page: 50, has_more: false, next_cursor: null },
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

  it('pages forward with the next cursor and back to the newest entries', async () => {
    const entry = {
      id: 2,
      actor_type: 'system',
      action: 'grant',
      resource: 'platform_operators',
      method: 'CLI',
      path: 'operator:grant',
      status_code: 200,
      created_at: '2026-09-30T04:40:00Z',
    };
    const page = (next: string | null) =>
      json({
        code: 0,
        message: 'success',
        data: [entry],
        meta: { per_page: 50, has_more: next !== null, next_cursor: next },
      });

    const first = renderWith(page('older-1'));
    fireEvent.click(await screen.findByRole('button', { name: 'Next' }));
    expect(first.onSearchChange).toHaveBeenLastCalledWith({ cursor: 'older-1' });
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    first.unmount();

    const deep = renderWith(page(null), { cursor: 'older-1' });
    expect(await screen.findByRole('button', { name: 'Next' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    expect(deep.onSearchChange).toHaveBeenLastCalledWith({ cursor: undefined });
  });
});
