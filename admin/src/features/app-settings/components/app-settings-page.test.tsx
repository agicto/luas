import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@/i18n';
import { AppSettingsPage } from '@/features/app-settings/components/app-settings-page';

const setting = {
  scope: 'app',
  key: 'app.support_email',
  kind: 'string',
  visibility: 'public',
  value: 'help@example.test',
  version: 3,
  source: 'override',
  updated_at: '2026-09-30T00:00:00Z',
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { 'content-type': 'application/json' },
    status,
  });
}

function renderPage(fetchMock: ReturnType<typeof vi.fn>) {
  vi.stubGlobal('fetch', fetchMock);
  render(
    <QueryClientProvider
      client={
        new QueryClient({
          defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
        })
      }
    >
      <AppSettingsPage />
    </QueryClientProvider>,
  );
}

describe('AppSettingsPage', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('saves with the loaded version as an If-Match precondition', async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation((_url: string, init: RequestInit) =>
        Promise.resolve(
          init.method === 'PATCH'
            ? json({
                code: 0,
                message: 'success',
                data: { ...setting, value: 'ops@example.test', version: 4 },
              })
            : json({ code: 0, message: 'success', data: [setting] }),
        ),
      );
    renderPage(fetchMock);

    fireEvent.change(await screen.findByLabelText('app.support_email'), {
      target: { value: 'ops@example.test' },
    });
    fireEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => {
      const patch = fetchMock.mock.calls.find(
        ([, init]) => (init as RequestInit).method === 'PATCH',
      );
      expect(patch?.[0]).toBe('/api/v1/operator/settings/app.support_email');
      expect(new Headers((patch?.[1] as RequestInit).headers).get('If-Match')).toBe('"setting-v3"');
      expect(JSON.parse(String((patch?.[1] as RequestInit).body))).toEqual({
        value: 'ops@example.test',
      });
    });
  });

  it('explains a concurrent change', async () => {
    renderPage(
      vi
        .fn()
        .mockImplementation((_url: string, init: RequestInit) =>
          Promise.resolve(
            init.method === 'PATCH'
              ? json({ code: 412, error_code: 'SETTING.VERSION_CONFLICT', message: 'stale' }, 412)
              : json({ code: 0, message: 'success', data: [setting] }),
          ),
        ),
    );

    fireEvent.change(await screen.findByLabelText('app.support_email'), {
      target: { value: 'x@example.test' },
    });
    fireEvent.click(screen.getByRole('button', { name: /save/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent(/changed this setting meanwhile/i);
  });

  it('explains when the setting starter is not active', async () => {
    renderPage(
      vi
        .fn()
        .mockResolvedValue(json({ code: 404, error_code: 'COMMON.NOT_FOUND', message: 'nf' }, 404)),
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(/setting starter/i);
  });
});
