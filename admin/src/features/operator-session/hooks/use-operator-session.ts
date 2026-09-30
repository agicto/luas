import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ApiErrorCode } from '@/http/codes';
import { ApiError, csrf } from '@/http/client';
import { operatorSessionService } from '@/features/operator-session/services/operator-session-service';
import type { OperatorSession, SignInInput } from '@/features/operator-session/types';

export const operatorSessionQueryKey = ['operator', 'session'] as const;

export function operatorSessionQueryOptions() {
  return queryOptions({
    queryKey: operatorSessionQueryKey,
    queryFn: async ({ signal }) => {
      const session = await operatorSessionService.current(signal);
      csrf.set(session.csrf_token);
      return session;
    },
    retry: false,
    staleTime: 60_000,
  });
}

/** Errors that mean the browser has no usable operator session and must sign in again. */
export function isSignedOut(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.status === 401 ||
      error.errorCode === ApiErrorCode.OPERATOR_FORBIDDEN ||
      error.errorCode === ApiErrorCode.AUTH_ACCOUNT_DISABLED)
  );
}

export function useOperatorSession() {
  return useQuery(operatorSessionQueryOptions());
}

export function useSignIn() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: SignInInput) => operatorSessionService.signIn(input),
    onSuccess: (session: OperatorSession) => {
      csrf.set(session.csrf_token);
      queryClient.setQueryData(operatorSessionQueryKey, session);
    },
  });
}

export function useSignOut() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => operatorSessionService.signOut(),
    onSettled: () => {
      csrf.set(undefined);
      queryClient.clear();
    },
  });
}

/** Refetches the session once so a stale CSRF token can be replaced before a retry. */
export async function recoverCsrfToken(): Promise<void> {
  const session = await operatorSessionService.current();
  csrf.set(session.csrf_token);
}
