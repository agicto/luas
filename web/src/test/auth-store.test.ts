import { describe, expect, it, vi } from 'vitest';

import { createAuthStore } from '@/features/auth/store/auth-store';
import type { AuthBootstrap, AuthUser } from '@/features/auth/types';

const ada: AuthUser = {
  id: 'user-ada',
  email: 'ada@example.com',
  name: 'Ada Lovelace',
  role: 'admin',
};

const grace: AuthUser = {
  id: 'user-grace',
  email: 'grace@example.com',
  name: 'Grace Hopper',
  role: 'member',
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

  it('settles failed client session resolution as unauthenticated', async () => {
    const loadCurrentUser = vi.fn().mockRejectedValue(new Error('unauthorized'));
    const { store } = createStore({ status: 'client-required' }, loadCurrentUser);

    await store.getState().initializeAuth();

    expect(store.getState()).toMatchObject({
      status: 'unauthenticated',
      user: null,
    });
  });
});
