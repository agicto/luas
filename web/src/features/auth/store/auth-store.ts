'use client';

import { createContext, useContext } from 'react';
import { useStore } from 'zustand';
import { createStore, type StoreApi } from 'zustand/vanilla';

import { authService } from '@/features/auth/services/auth-service';
import type {
  AuthBootstrap,
  AuthResponse,
  AuthStatus,
  AuthUser,
} from '@/features/auth/types';

export interface AuthState {
  user: AuthUser | null;
  status: AuthStatus;
  setUser: (user: AuthUser | null) => void;
  reset: () => void;
  initializeAuth: () => Promise<void>;
}

type CurrentUserLoader = () => Promise<AuthResponse>;
export type AuthStore = StoreApi<AuthState>;

function initialAuthState(
  bootstrap: AuthBootstrap
): Pick<AuthState, 'status' | 'user'> {
  switch (bootstrap.status) {
    case 'authenticated':
      return { status: 'authenticated', user: bootstrap.user };
    case 'unauthenticated':
      return { status: 'unauthenticated', user: null };
    case 'client-required':
      return { status: 'idle', user: null };
  }
}

/**
 * Creates one auth store per provider instance so server-rendered users never
 * leak through a module-level singleton into another request.
 */
export function createAuthStore(
  bootstrap: AuthBootstrap,
  loadCurrentUser: CurrentUserLoader = authService.me
): AuthStore {
  let initialization: Promise<void> | null = null;

  return createStore<AuthState>()((set, get) => ({
    ...initialAuthState(bootstrap),

    setUser: (user) =>
      set({
        status: user ? 'authenticated' : 'unauthenticated',
        user,
      }),

    reset: () => set({ status: 'unauthenticated', user: null }),

    initializeAuth: () => {
      if (initialization) {
        return initialization;
      }

      if (get().status !== 'idle') {
        return Promise.resolve();
      }

      set({ status: 'loading' });
      initialization = loadCurrentUser()
        .then(({ user }) => {
          set({ status: 'authenticated', user });
        })
        .catch(() => {
          set({ status: 'unauthenticated', user: null });
        })
        .finally(() => {
          initialization = null;
        });

      return initialization;
    },
  }));
}

export const AuthStoreContext = createContext<AuthStore | null>(null);

function useAuthStoreSelector<T>(selector: (state: AuthState) => T): T {
  const store = useContext(AuthStoreContext);

  if (!store) {
    throw new Error('useAuthStore must be used within AuthProvider');
  }

  return useStore(store, selector);
}

function useUser() {
  return useAuthStoreSelector((state) => state.user);
}

function useStatus() {
  return useAuthStoreSelector((state) => state.status);
}

function useAuthenticatedFlag() {
  return useAuthStoreSelector((state) => state.status === 'authenticated');
}

function useIsLoading() {
  return useAuthStoreSelector(
    (state) => state.status === 'idle' || state.status === 'loading'
  );
}

function useIsSystemReady() {
  return useAuthStoreSelector(
    (state) => state.status !== 'idle' && state.status !== 'loading'
  );
}

function useSetUser() {
  return useAuthStoreSelector((state) => state.setUser);
}

function useReset() {
  return useAuthStoreSelector((state) => state.reset);
}

function useInitializeAuth() {
  return useAuthStoreSelector((state) => state.initializeAuth);
}

/**
 * Stable selector surface retained for feature consumers.
 */
export const useAuthStore = {
  use: {
    user: useUser,
    status: useStatus,
    isAuthenticated: useAuthenticatedFlag,
    isLoading: useIsLoading,
    isSystemReady: useIsSystemReady,
    setUser: useSetUser,
    reset: useReset,
    initializeAuth: useInitializeAuth,
  },
};

export const useCurrentUser = () => useAuthStore.use.user();
export const useIsAuthenticated = () => useAuthStore.use.isAuthenticated();
export const useAuthLoading = () => useAuthStore.use.isLoading();
