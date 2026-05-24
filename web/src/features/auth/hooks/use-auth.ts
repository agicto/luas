'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { authConfig } from '@/config/auth';
import { authService } from '@/features/auth/services/auth-service';
import { useAuthStore } from '@/features/auth/store/auth-store';
import type { LoginRequest, RegisterRequest } from '@/features/auth/types';
import { useT } from '@/i18n';

function resolveReturnUrl(value: string | null): string {
  if (!value || !value.startsWith('/')) {
    return authConfig.routes.afterLogin;
  }

  return value;
}

/**
 * @hook useLogin
 * @description Runs the mock login flow and syncs the auth store with the session cookie.
 */
export function useLogin() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const setUser = useAuthStore.use.setUser();
  const t = useT();

  return useMutation({
    mutationFn: (data: LoginRequest) => authService.login(data),
    onSuccess: ({ user }) => {
      setUser(user);
      toast.success(t('auth.loginSuccess'));
      router.replace(resolveReturnUrl(searchParams.get('returnUrl')));
    },
  });
}

/**
 * @hook useRegister
 * @description Registers a mock account, creates the session cookie, and redirects into the console.
 */
export function useRegister() {
  const router = useRouter();
  const setUser = useAuthStore.use.setUser();
  const t = useT();

  return useMutation({
    mutationFn: (data: RegisterRequest) => authService.register(data),
    onSuccess: ({ user }) => {
      setUser(user);
      toast.success(t('auth.accountCreated'));
      router.replace(authConfig.routes.afterLogin);
    },
  });
}

/**
 * @hook useLogout
 * @description Clears the mock session cookie and local auth state.
 */
export function useLogout() {
  const router = useRouter();
  const reset = useAuthStore.use.reset();
  const t = useT();

  return useMutation({
    mutationFn: () => authService.logout(),
    onSuccess: () => {
      reset();
      toast.success(t('auth.logoutSuccess'));
      router.replace(authConfig.routes.afterLogout);
    },
  });
}
