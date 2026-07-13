/**
 * Example React Query hooks
 * Encapsulating state, side effects, and cache management.
 */

'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { exampleService } from '@/features/example/services/example-service';
import type {
  CreateExampleRequest,
  ExampleItem,
  ExampleListResponse,
  ExampleQuerySchema,
  UpdateExampleRequest,
} from '@/features/example/types';
import { useT } from '@/i18n';

// ============================================================================
// Query Keys
// ============================================================================

/**
 * Centralized query keys for the example domain
 */
export const exampleKeys = {
  all: ['examples'] as const,
  lists: () => [...exampleKeys.all, 'list'] as const,
  list: (params: ExampleQuerySchema) => [...exampleKeys.lists(), params] as const,
  details: () => [...exampleKeys.all, 'detail'] as const,
  detail: (id: string) => [...exampleKeys.details(), id] as const,
};

function buildOptimisticExampleItem(data: CreateExampleRequest): ExampleItem {
  const timestamp = new Date().toISOString();

  return {
    id: `temp-${crypto.randomUUID()}`,
    title: data.title,
    description: data.description,
    status: data.status ?? 'active',
    createdAt: timestamp,
    updatedAt: timestamp,
  };
}

function updateListResponse(
  old: ExampleListResponse | undefined,
  updater: (items: ExampleItem[]) => ExampleItem[]
): ExampleListResponse {
  const currentItems = old?.items ?? [];
  const nextItems = updater(currentItems);

  return {
    items: nextItems,
    total: old ? Math.max(nextItems.length, old.total + (nextItems.length - currentItems.length)) : nextItems.length,
    page: old?.page ?? 1,
    pageSize: old?.pageSize ?? Math.max(nextItems.length, 1),
  };
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for fetching example items list
 */
export function useExamples(params?: ExampleQuerySchema) {
  return useQuery({
    queryKey: exampleKeys.list(params || {}),
    queryFn: () => exampleService.getList(params),
  });
}

/**
 * Hook for fetching example item detail
 */
export function useExample(id: string) {
  return useQuery({
    queryKey: exampleKeys.detail(id),
    queryFn: () => exampleService.getDetail(id),
    enabled: !!id,
  });
}

/**
 * Hook for creating a new example item
 * Implements Optimistic Updates (List appending)
 */
export function useCreateExample() {
  const queryClient = useQueryClient();
  const t = useT();

  return useMutation({
    mutationFn: (data: CreateExampleRequest) => exampleService.create(data),
    meta: LOCAL_ERROR_HANDLING_META,

    onMutate: async (newItem) => {
      await queryClient.cancelQueries({ queryKey: exampleKeys.lists() });
      const optimisticItem = buildOptimisticExampleItem(newItem);

      queryClient.setQueriesData<ExampleListResponse>({ queryKey: exampleKeys.lists() }, (old) => {
        return updateListResponse(old, (items) => [optimisticItem, ...items]);
      });
    },

    onError: () => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
      toast.error(t.errors('serverError'));
    },

    onSettled: (data, error) => {
      // Final synchronization
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
      if (data && !error) {
        toast.success(t.common('success'));
      }
    },
  });
}

/**
 * Hook for updating an existing example item
 * Implements Optimistic Updates with "Refetch-on-Failure" rollback strategy.
 */
export function useUpdateExample() {
  const queryClient = useQueryClient();
  const t = useT();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateExampleRequest }) =>
      exampleService.update(id, data),
    meta: LOCAL_ERROR_HANDLING_META,

    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: exampleKeys.detail(id) });
      const previousItem = queryClient.getQueryData(exampleKeys.detail(id));

      if (previousItem) {
        queryClient.setQueryData<ExampleItem>(exampleKeys.detail(id), (old) => {
          if (!old) {
            return old;
          }

          return {
            ...old,
            ...data,
          };
        });
      }

      return { previousItem };
    },

    onError: (_error, { id }) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      toast.error(t.errors('serverError'));
    },

    onSettled: (data, error, { id }) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
      
      if (data && !error) {
        toast.success(t.common('success'));
      }
    },
  });
}

/**
 * Hook for deleting an example item
 * Implements Optimistic Updates (List filtering)
 */
export function useDeleteExample() {
  const queryClient = useQueryClient();
  const t = useT();

  return useMutation({
    mutationFn: (id: string) => exampleService.delete(id),
    meta: LOCAL_ERROR_HANDLING_META,

    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: exampleKeys.lists() });
      await queryClient.cancelQueries({ queryKey: exampleKeys.detail(id) });

      queryClient.setQueriesData<ExampleListResponse>({ queryKey: exampleKeys.lists() }, (old) => {
        return updateListResponse(old, (items) => items.filter((item) => item.id !== id));
      });

      return { id };
    },

    onError: (_error, id) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      toast.error(t.errors('serverError'));
    },

    onSettled: (data, error, id) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      
      if (!error) {
        toast.success(t.common('success'));
      }
    },
  });
}
