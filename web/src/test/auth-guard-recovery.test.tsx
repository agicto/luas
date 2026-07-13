import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import {
  NextIntlClientProvider,
  type AbstractIntlMessages,
} from 'next-intl';
import { describe, expect, it, vi } from 'vitest';

import { AuthGuard } from '@/features/auth/components/auth-guard';
import {
  AuthStoreContext,
  createAuthStore,
} from '@/features/auth/store/auth-store';
import type { AuthUser } from '@/features/auth/types';
import { messages } from '@/i18n/modules';

const user: AuthUser = {
  id: 'user-ada',
  email: 'ada@example.com',
  name: 'Ada Lovelace',
  role: 'admin',
};

function renderGuard(
  status: 'forbidden' | 'loading' | 'unavailable',
  loadCurrentUser = vi.fn().mockResolvedValue({ user })
) {
  const store = createAuthStore({ status: 'client-required' }, loadCurrentUser);
  store.setState({ status, user: null });

  render(
    <NextIntlClientProvider
      locale="en-US"
      messages={messages as unknown as AbstractIntlMessages}
    >
      <AuthStoreContext.Provider value={store}>
        <AuthGuard>
          <div>Protected content</div>
        </AuthGuard>
      </AuthStoreContext.Provider>
    </NextIntlClientProvider>
  );

  return { loadCurrentUser, store };
}

describe('AuthGuard recovery states', () => {
  it('renders one busy main landmark while resolving the session', () => {
    renderGuard('loading');

    const main = screen.getByRole('main');
    expect(main).toHaveAttribute('aria-busy', 'true');
    expect(screen.getAllByRole('main')).toHaveLength(1);
  });

  it('renders a non-redirecting forbidden state', () => {
    renderGuard('forbidden');

    expect(screen.getAllByRole('main')).toHaveLength(1);
    expect(screen.getByRole('alert')).toHaveTextContent(
      "You don't have permission to access this resource"
    );
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    expect(screen.queryByText('Protected content')).not.toBeInTheDocument();
  });

  it('retries an unavailable session and restores protected content', async () => {
    const { loadCurrentUser } = renderGuard('unavailable');

    expect(screen.getAllByRole('main')).toHaveLength(1);
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Unable to verify your session'
    );
    expect(screen.queryByText('Protected content')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => {
      expect(screen.getByText('Protected content')).toBeInTheDocument();
    });
    expect(loadCurrentUser).toHaveBeenCalledTimes(1);
  });
});
