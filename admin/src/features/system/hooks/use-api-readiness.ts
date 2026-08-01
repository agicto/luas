import { useQuery } from '@tanstack/react-query';
import { systemService } from '@/features/system/services/system-service';

export const readinessQueryKey = ['system', 'readiness'] as const;

export function useApiReadiness() {
  return useQuery({
    queryKey: readinessQueryKey,
    queryFn: ({ signal }) => systemService.readiness(signal),
    refetchInterval: false,
    staleTime: 15_000,
  });
}
