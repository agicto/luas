import { auditService } from '@/features/audit/services/audit-service';

const emptyPage = {
  code: 0,
  message: 'success',
  data: [],
  meta: { current_page: 1, last_page: 1, per_page: 50, total: 0 },
};

describe('auditService.list', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('turns inclusive local days into an exclusive instant range', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(emptyPage), {
        headers: { 'content-type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await auditService.list({
      from: '2026-09-01',
      to: '2026-09-30',
      user_id: 7,
      action: 'disable',
    });

    const url = new URL(String(fetchMock.mock.calls[0]?.[0]), 'http://localhost');
    expect(url.pathname).toBe('/api/v1/operator/audit-logs');
    expect(url.searchParams.get('from')).toBe(new Date(2026, 8, 1).toISOString());
    expect(url.searchParams.get('to')).toBe(new Date(2026, 9, 1).toISOString());
    expect(url.searchParams.get('user_id')).toBe('7');
    expect(url.searchParams.get('action')).toBe('disable');
  });
});
