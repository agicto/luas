'use client';

import { useMutation } from '@tanstack/react-query';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { authConfig } from '@/config/auth';
import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { authService } from '@/features/auth/services/auth-service';
import { useAuthStore } from '@/features/auth/store/auth-store';
import { isUnauthenticatedAuthError, resolveAuthErrorKey } from '@/features/auth/utils/auth-error';
import { resolveReturnUrl } from '@/features/auth/utils/return-url';
import type { LoginRequest, RegisterRequest } from '@/features/auth/types';
import { useT } from '@/i18n';

/**
 * @hook useLogin
 * @description Runs the configured login contract and redirects after validated success.
 */
export function useLogin() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const t = useT();

  return useMutation({
    mutationFn: (data: LoginRequest) => authService.login(data),
    meta: LOCAL_ERROR_HANDLING_META,
    retry: false,
    onSuccess: () => {
      toast.success(t('auth.loginSuccess'));
      router.replace(resolveReturnUrl(searchParams.get('returnUrl')));
    },
  });
}

/**
 * @hook useRegister
 * @description Runs the configured registration contract and redirects after validated success.
 */
export function useRegister() {
  const router = useRouter();
  const t = useT();

  return useMutation({
    mutationFn: (data: RegisterRequest) => authService.register(data),
    meta: LOCAL_ERROR_HANDLING_META,
    retry: false,
    onSuccess: () => {
      toast.success(t('auth.accountCreated'));
      router.replace(authConfig.routes.afterLogin);
    },
  });
}

/**
 * @hook useLogout
 * @description Ends the configured session and reconciles local auth state idempotently.
 */
export function useLogout() {
  const router = useRouter();
  const reset = useAuthStore.use.reset();
  const t = useT();

  const completeLogout = () => {
    reset();
    router.replace(authConfig.routes.afterLogout);
  };

  return useMutation({
    mutationFn: () => authService.logout(),
    meta: LOCAL_ERROR_HANDLING_META,
    retry: false,
    onSuccess: () => {
      completeLogout();
      toast.success(t('auth.logoutSuccess'));
    },
    onError: error => {
      if (isUnauthenticatedAuthError(error)) {
        completeLogout();
        toast.success(t('auth.logoutSuccess'));
        return;
      }

      toast.error(t(resolveAuthErrorKey(error, 'logout')));
    },
  });
}
