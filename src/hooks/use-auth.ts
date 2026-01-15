'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { authApi } from '@/services/auth';
import { useAuthStore } from '@/store/auth-store';
import { authConfig } from '@/config/auth';
import { useT } from '@/i18n';
import type { RegisterRequest } from '@/types/auth';

// ============================================================================
// Auth Query Keys
// ============================================================================

export const authKeys = {
  all: ['auth'] as const,
  user: () => [...authKeys.all, 'user'] as const,
  session: () => [...authKeys.all, 'session'] as const,
};

// ============================================================================
// Login Mutation
// ============================================================================

interface LoginParams {
  email: string;
  password: string;
}

/**
 * Login mutation hook with toast notifications
 * 
 * @example
 * ```tsx
 * const { mutate: login, isPending } = useLogin();
 * login({ email, password });
 * ```
 */
export function useLogin() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const setUser = useAuthStore.use.setUser();
  const t = useT();

  return useMutation({
    mutationFn: async ({ email, password }: LoginParams) => {
      return authApi.login({ email, password });
    },

    onSuccess: (user) => {
      setUser(user);
      toast.success(t.auth('loginSuccess'), {
        description: t.auth('welcomeBack', { name: user.name }),
      });
      queryClient.invalidateQueries({ queryKey: authKeys.user() });
      router.push(authConfig.routes.afterLogin);
    },
  });
}

// ============================================================================
// Register Mutation
// ============================================================================

interface RegisterParams {
  name: string;
  email: string;
  password: string;
}

/**
 * Register mutation hook with toast notifications
 * 
 * @example
 * ```tsx
 * const { mutate: register, isPending } = useRegister();
 * register({ name, email, password });
 * ```
 */
export function useRegister() {
  const t = useT();
  const loginMutation = useLogin();

  return useMutation({
    mutationFn: async ({ name, email, password }: RegisterParams) => {
      const data: RegisterRequest = { name, email, password };
      return authApi.register(data);
    },

    onSuccess: (_, variables) => {
      toast.success(t.auth('registerSuccess'), {
        description: t.auth('accountCreated'),
      });
      // Auto-login after successful registration
      loginMutation.mutate({ 
        email: variables.email, 
        password: variables.password 
      });
    },
  });
}

// ============================================================================
// Logout Mutation
// ============================================================================

/**
 * Logout mutation hook with toast notifications
 * 
 * @example
 * ```tsx
 * const { mutate: logout, isPending } = useLogout();
 * logout();
 * ```
 */
export function useLogout() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const setUser = useAuthStore.use.setUser();
  const t = useT();

  return useMutation({
    mutationFn: async () => {
      return authApi.logout();
    },

    onSuccess: () => {
      setUser(null);
      queryClient.clear();
      toast.success(t.auth('accountCreated'));
      router.push(authConfig.routes.afterLogout);
    },

    onError: () => {
      // Still logout locally even if API fails
      setUser(null);
      queryClient.clear();
      router.push(authConfig.routes.afterLogout);
    },
  });
}
