import { create } from 'zustand';
import { authService } from '@/features/auth/services/auth-service';
import type { AuthUser } from '@/features/auth/types';
import { createSelectors } from '@/store/utils/selectors';

interface AuthState {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isSystemReady: boolean;

  setUser: (user: AuthUser | null) => void;
  reset: () => void;
  initializeAuth: () => Promise<void>;
}

const defaultState = {
  user: null as AuthUser | null,
  isAuthenticated: false,
  isLoading: false,
  isSystemReady: false,
};

/**
 * Authentication store backed by the server session cookie.
 */
const useAuthStoreBase = create<AuthState>()((set) => ({
  ...defaultState,

  setUser: (user) =>
    set({
      user,
      isAuthenticated: Boolean(user),
    }),

  reset: () => set(defaultState),

  initializeAuth: async () => {
    set({ isLoading: true });

    try {
      const { user } = await authService.me();
      set({
        user,
        isAuthenticated: true,
        isSystemReady: true,
      });
    } catch {
      set({
        user: null,
        isAuthenticated: false,
        isSystemReady: true,
      });
    } finally {
      set({ isLoading: false });
    }
  },
}));

/**
 * Auth store with selectors for optimized component updates
 */
export const useAuthStore = createSelectors(useAuthStoreBase);

export const useCurrentUser = () => useAuthStore.use.user();
export const useIsAuthenticated = () => useAuthStore.use.isAuthenticated();
export const useAuthLoading = () => useAuthStore.use.isLoading();
