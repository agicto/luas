import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { csrf } from '@/http/client';
import '@/i18n';
import { SignInPage } from '@/features/operator-session/components/sign-in-page';

const navigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigate }));

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <SignInPage redirectTo="/console/users" />
    </QueryClientProvider>,
  );
  return queryClient;
}

function respond(body: unknown, status: number) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify(body), {
        headers: { 'content-type': 'application/json' },
        status,
      }),
    ),
  );
}

function submit() {
  fireEvent.change(screen.getByLabelText(/username or email/i), { target: { value: 'ops' } });
  fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } });
  fireEvent.click(screen.getByRole('button', { name: /sign in/i }));
}

describe('SignInPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    navigate.mockReset();
    csrf.set(undefined);
  });

  it('keeps the CSRF token in memory and navigates to the requested console page', async () => {
    respond(
      {
        code: 0,
        message: 'success',
        data: {
          operator: { id: 1, username: 'ops', email: 'ops@example.test', nickname: 'Ops' },
          csrf_token: 'csrf-token-0123456789',
        },
      },
      200,
    );
    const queryClient = renderPage();

    submit();

    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({ to: '/console/users', replace: true }),
    );
    expect(csrf.get()).toBe('csrf-token-0123456789');
    expect(queryClient.getQueryData(['operator', 'session'])).toMatchObject({
      operator: { username: 'ops' },
    });
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
  });

  it('explains that a verified account lacks operator access', async () => {
    respond(
      { code: 403, error_code: 'OPERATOR.FORBIDDEN', message: 'Operator access is required' },
      403,
    );
    renderPage();

    submit();

    expect(await screen.findByRole('alert')).toHaveTextContent(/platform-operator access/i);
    expect(navigate).not.toHaveBeenCalled();
  });
});
