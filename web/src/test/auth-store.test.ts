import { describe, expect, it, vi } from 'vitest';

import { createAuthStore } from '@/features/auth/store/auth-store';
import type { AuthBootstrap, AuthUser } from '@/features/auth/types';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

const ada: AuthUser = {
  id: 'user-ada',
  email: 'ada@example.com',
  name: 'Ada Lovelace',
};

const grace: AuthUser = {
  id: 'user-grace',
  email: 'grace@example.com',
  name: 'Grace Hopper',
};

function createStore(
  bootstrap: AuthBootstrap,
  loadCurrentUser = vi.fn().mockResolvedValue({ user: ada })
) {
  return {
    loadCurrentUser,
    store: createAuthStore(bootstrap, loadCurrentUser),
  };
}

describe('auth store bootstrap', () => {
  it('starts authenticated without a client request when the server resolved the session', async () => {
    const { loadCurrentUser, store } = createStore({
      status: 'authenticated',
      user: ada,
    });

    expect(store.getState()).toMatchObject({
      status: 'authenticated',
      user: ada,
    });

    await store.getState().initializeAuth();

    expect(loadCurrentUser).not.toHaveBeenCalled();
  });

  it('keeps server-authenticated users isolated between provider instances', () => {
    const first = createStore({ status: 'authenticated', user: ada }).store;
    const second = createStore({ status: 'authenticated', user: grace }).store;

    expect(first.getState().user).toEqual(ada);
    expect(second.getState().user).toEqual(grace);
  });

  it('starts ready and unauthenticated when the server rejects the mock session', async () => {
    const { loadCurrentUser, store } = createStore({ status: 'unauthenticated' });

    expect(store.getState()).toMatchObject({
      status: 'unauthenticated',
      user: null,
    });

    await store.getState().initializeAuth();

    expect(loadCurrentUser).not.toHaveBeenCalled();
  });

  it.each(['forbidden', 'unavailable'] as const)(
    'preserves a server-resolved %s state and retries through the browser seam',
    async (status) => {
      const loadCurrentUser = vi.fn().mockResolvedValue({ user: ada });
      const store = createAuthStore({ status }, loadCurrentUser);

      expect(store.getState()).toMatchObject({ status, user: null });
      await store.getState().initializeAuth();

      expect(loadCurrentUser).toHaveBeenCalledTimes(1);
      expect(store.getState()).toMatchObject({
        status: 'authenticated',
        user: ada,
      });
    }
  );

  it('deduplicates concurrent client session resolution', async () => {
    let resolveRequest: ((value: { user: AuthUser }) => void) | undefined;
    const loadCurrentUser = vi.fn(
      () =>
        new Promise<{ user: AuthUser }>((resolve) => {
          resolveRequest = resolve;
        })
    );
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    const first = store.getState().initializeAuth();
    const second = store.getState().initializeAuth();

    expect(loadCurrentUser).toHaveBeenCalledTimes(1);
    expect(store.getState().status).toBe('loading');

    resolveRequest?.({ user: ada });
    await Promise.all([first, second]);

    expect(store.getState()).toMatchObject({
      status: 'authenticated',
      user: ada,
    });
  });

  it('treats only an unauthorized response as an absent session', async () => {
    const loadCurrentUser = vi
      .fn()
      .mockRejectedValue(
        new ApiError('Session expired', ApiErrorCode.AUTH_UNAUTHORIZED, 401)
      );
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'unauthenticated',
      user: null,
    });
  });

  it('keeps a forbidden session distinct from an absent session', async () => {
    const loadCurrentUser = vi
      .fn()
      .mockRejectedValue(
        new ApiError('Access denied', ApiErrorCode.AUTH_FORBIDDEN, 403)
      );
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'forbidden',
      user: null,
    });
  });

  it('prefers canonical auth evidence over a conflicting status fallback', async () => {
    const loadCurrentUser = vi
      .fn()
      .mockRejectedValue(
        new ApiError(
          'Account disabled',
          ApiErrorCode.AUTH_ACCOUNT_DISABLED,
          401
        )
      );
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'forbidden',
      user: null,
    });
  });

  it.each([
    new Error('unexpected failure'),
    new ApiError('Network unavailable', ClientErrorCode.NETWORK_ERROR),
    new ApiError(
      'Service unavailable',
      ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      503
    ),
  ])('preserves unknown and transient failures as unavailable', async (error) => {
    const loadCurrentUser = vi.fn().mockRejectedValue(error);
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'unavailable',
      user: null,
    });
  });

  it.each([
    {},
    { user: null },
    {
      user: {
        id: 'user-ada',
        email: 'ada@example.com',
        name: '',
      },
    },
  ])('rejects a malformed successful session payload', async (payload) => {
    const loadCurrentUser = vi.fn().mockResolvedValue(payload);
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'unavailable',
      user: null,
    });
  });

  it('retries an unavailable session without duplicating requests', async () => {
    const loadCurrentUser = vi
      .fn()
      .mockRejectedValueOnce(
        new ApiError('Timed out', ClientErrorCode.TIMEOUT)
      )
      .mockResolvedValueOnce({ user: ada });
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();
    expect(store.getState().status).toBe('unavailable');

    const firstRetry = store.getState().initializeAuth();
    const secondRetry = store.getState().initializeAuth();
    await Promise.all([firstRetry, secondRetry]);

    expect(loadCurrentUser).toHaveBeenCalledTimes(2);
    expect(store.getState()).toMatchObject({
      status: 'authenticated',
      user: ada,
    });
  });
});
