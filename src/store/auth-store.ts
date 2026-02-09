// Authentication state management (Template)
// Demonstrates Zustand usage for core application state.

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { createSelectors } from './utils/selectors';

/**
 * Placeholder User type
 */
export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
}

interface AuthState {
  // Core auth state
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  accessToken: string | null;
  
  // System state
  isSystemReady: boolean;
  
  // Actions
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  
  // Global actions used during app startup
  initializeAuth: () => Promise<void>;
  
  // Reset
  reset: () => void;
}

const defaultState = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
  accessToken: null,
  isSystemReady: false,
};

/**
 * Authentication store with persistence
 */
const useAuthStoreBase = create<AuthState>()(
  persist(
    (set) => ({
      ...defaultState,
      
      setUser: (user) => set({ 
        user, 
        isAuthenticated: !!user,
        accessToken: user ? (user as any).accessToken : null,
      }),
      
      setLoading: (isLoading) => set({ isLoading }),
      
      initializeAuth: async () => {
        set({ isLoading: true });
        try {
          // Placeholder for auth initialization logic
          // In a real app, verify tokens or fetch profile here
          set({ isSystemReady: true });
        } catch (error) {
          console.error('Auth initialization failed:', error);
          set({ isSystemReady: false });
        } finally {
          set({ isLoading: false });
        }
      },
      
      reset: () => set(defaultState),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        accessToken: state.accessToken,
        isSystemReady: state.isSystemReady,
      }),
    }
  )
);

/**
 * Access tokens outside of React components/hooks
 */
export const getAuthTokens = () => {
  const state = useAuthStoreBase.getState();
  return {
    accessToken: state.accessToken,
  };
};

export const setAuthTokens = (accessToken: string | null) => {
  useAuthStoreBase.setState({ accessToken });
};

/**
 * Auth store with selectors for optimized component updates
 */
export const useAuthStore = createSelectors(useAuthStoreBase);

// Convenience hooks for common patterns
export const useCurrentUser = () => useAuthStore.use.user();
export const useIsAuthenticated = () => useAuthStore.use.isAuthenticated();
export const useAuthLoading = () => useAuthStore.use.isLoading();
