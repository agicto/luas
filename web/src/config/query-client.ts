import { QueryClient, QueryCache, MutationCache } from '@tanstack/react-query';
import { hasLocalErrorHandling } from '@/config/query-meta';
import { handleError } from '@/http/error-handler';

/** Creates an isolated QueryClient for one provider instance. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => {
        if (!hasLocalErrorHandling(query.meta)) {
          handleError(error);
        }
      },
    }),
    mutationCache: new MutationCache({
      onError: (error, _variables, _onMutateResult, mutation) => {
        if (!hasLocalErrorHandling(mutation.meta)) {
          handleError(error);
        }
      },
    }),
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
        staleTime: 5 * 60 * 1000,
      },
      mutations: {
        // Write retries require endpoint-specific idempotency evidence.
        retry: false,
      },
    },
  });
}
