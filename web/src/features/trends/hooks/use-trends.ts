'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { useT } from '@/i18n';
import { trendService } from '@/features/trends/services/trend-service';
import type { TrendQuerySchema } from '@/features/trends/types';

export const trendKeys = {
  all: ['trends'] as const,
  lists: () => [...trendKeys.all, 'list'] as const,
  list: (params: TrendQuerySchema) => [...trendKeys.lists(), params] as const,
  summary: () => [...trendKeys.all, 'summary'] as const,
};

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

export function useTrends(params?: TrendQuerySchema) {
  return useQuery({
    queryKey: trendKeys.list(params ?? {}),
    queryFn: () => trendService.getList(params),
  });
}

export function useTrendSummary() {
  return useQuery({
    queryKey: trendKeys.summary(),
    queryFn: () => trendService.getSummary(),
  });
}

export function useCreateTrendSyncRun() {
  const queryClient = useQueryClient();
  const t = useT('trends');

  return useMutation({
    mutationFn: () => trendService.createSyncRun(),
    onSuccess: (result) => {
      toast.success(t('toast.syncSuccess', {
        inserted: result.inserted,
        candidates: result.candidates,
      }));
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('errors.sync')));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: trendKeys.all });
    },
  });
}
