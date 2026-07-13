import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useLogout } from '@/features/auth/hooks/use-auth';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  replace: vi.fn(),
  reset: vi.fn(),
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('@/features/auth/services/auth-service', () => ({
  authService: {
    logout: mocks.logout,
  },
}));

vi.mock('@/features/auth/store/auth-store', () => ({
  useAuthStore: {
    use: {
      reset: () => mocks.reset,
    },
  },
}));

vi.mock('@/i18n', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace: mocks.replace }),
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock('sonner', () => ({ toast: mocks.toast }));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('logout mutation recovery', () => {
  beforeEach(() => {
    mocks.logout.mockReset();
    mocks.replace.mockReset();
    mocks.reset.mockReset();
    mocks.toast.error.mockReset();
    mocks.toast.success.mockReset();
  });

  it('completes logout when the server says the session is already absent', async () => {
    mocks.logout.mockRejectedValueOnce(
      new ApiError('raw unauthorized message', ApiErrorCode.AUTH_UNAUTHORIZED, 401)
    );
    const { result } = renderHook(() => useLogout(), {
      wrapper: createWrapper(),
    });

    act(() => result.current.mutate());

    await waitFor(() => expect(mocks.reset).toHaveBeenCalledTimes(1));
    expect(mocks.replace).toHaveBeenCalledWith('/login');
    expect(mocks.toast.success).toHaveBeenCalledWith('auth.logoutSuccess');
    expect(mocks.toast.error).not.toHaveBeenCalled();
  });

  it('preserves local state when logout availability is unknown', async () => {
    mocks.logout.mockRejectedValueOnce(
      new ApiError('raw network message', ClientErrorCode.NETWORK_ERROR)
    );
    const { result } = renderHook(() => useLogout(), {
      wrapper: createWrapper(),
    });

    act(() => result.current.mutate());

    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith('errors.networkError'));
    expect(mocks.reset).not.toHaveBeenCalled();
    expect(mocks.replace).not.toHaveBeenCalled();
    expect(mocks.toast.success).not.toHaveBeenCalled();
  });
});
