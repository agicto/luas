'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { taskApi } from '@/services/task';
import type { Task, TaskCreateDto, TaskUpdateDto, TaskListResponse } from '@/services/task';
import { toast } from 'sonner';

/**
 * Gold Standard CRUD Example - Task Management
 * 
 * This example demonstrates:
 * 1. Query Key Factories for safe invalidation.
 * 2. Contract-based types from generated schema.
 * 3. Zod-validated service calls.
 * 4. Optimistic updates for high-performance UX.
 * 5. Global Error Handling integration (automated toasts).
 */

// ============================================================================
// Query Key Factory
// ============================================================================

export const taskKeys = {
  all: ['tasks'] as const,
  lists: () => [...taskKeys.all, 'list'] as const,
  list: (status?: Task['status']) => [...taskKeys.lists(), { status }] as const,
  details: () => [...taskKeys.all, 'detail'] as const,
  detail: (id: string) => [...taskKeys.details(), id] as const,
};

// ============================================================================
// Queries
// ============================================================================

/**
 * Hook to fetch tasks list
 */
export function useTasks(status?: Task['status']) {
  return useQuery({
    queryKey: taskKeys.list(status),
    queryFn: () => taskApi.list({ status }),
  });
}

/**
 * Hook to fetch task detail
 */
export function useTask(id: string) {
  return useQuery({
    queryKey: taskKeys.detail(id),
    queryFn: () => taskApi.get(id),
    enabled: !!id,
  });
}

// ============================================================================
// Mutations (Optimistic Updates)
// ============================================================================

/**
 * Mutation to create a task with optimistic update
 */
export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: TaskCreateDto) => taskApi.create(data),
    
    onMutate: async (newTask) => {
      // 1. Cancel outgoing refetchs
      await queryClient.cancelQueries({ queryKey: taskKeys.lists() });
      
      // 2. Snapshot current state
      const previousTasks = queryClient.getQueriesData<TaskListResponse>({ queryKey: taskKeys.lists() });
      
      // 3. Create optimistic task
      const optimisticTask: Task = {
        id: `temp-${Date.now()}`,
        title: newTask.title,
        status: newTask.status || 'todo',
        priority: newTask.priority || 'medium',
        description: newTask.description || '',
        createdAt: new Date().toISOString(),
      };
      
      // 4. Update cache
      queryClient.setQueriesData<TaskListResponse>(
        { queryKey: taskKeys.lists() },
        (old) => {
          if (!old || !old.data) return old;
          return {
            ...old,
            data: [optimisticTask, ...old.data],
            total: (old.total ?? 0) + 1,
          };
        }
      );
      
      return { previousTasks };
    },
    
    onError: (err, _, context) => {
      // Rollback to previous state
      if (context?.previousTasks) {
        context.previousTasks.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
      // Note: Global Error Handler will automatically show a toast. 
      // We only need to rollback state here.
    },
    
    onSuccess: (data) => {
      toast.success('Task created successfully', {
        description: `"${data.title}" has been added to the list.`,
      });
    },
    
    onSettled: () => {
      // Invalidate to ensures UI is in sync with server
      queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}

/**
 * Mutation to update a task with optimistic update
 */
export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: TaskUpdateDto }) => 
      taskApi.update(id, data),
    
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: taskKeys.detail(id) });
      await queryClient.cancelQueries({ queryKey: taskKeys.lists() });
      
      const previousDetail = queryClient.getQueryData<Task>(taskKeys.detail(id));
      const previousLists = queryClient.getQueriesData<TaskListResponse>({ queryKey: taskKeys.lists() });
      
      // Update detail cache
      if (previousDetail) {
        queryClient.setQueryData<Task>(taskKeys.detail(id), {
          ...previousDetail,
          ...data,
        });
      }
      
      // Update lists cache
      queryClient.setQueriesData<TaskListResponse>(
        { queryKey: taskKeys.lists() },
        (old) => {
          if (!old || !old.data) return old;
          return {
            ...old,
            data: old.data.map((task) => 
              task.id === id ? { ...task, ...data } : task
            ),
          };
        }
      );
      
      return { previousDetail, previousLists };
    },
    
    onError: (err, { id }, context) => {
      if (context?.previousDetail) {
        queryClient.setQueryData(taskKeys.detail(id), context.previousDetail);
      }
      if (context?.previousLists) {
        context.previousLists.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
    },
    
    onSuccess: (data) => {
      toast.success('Task updated');
    },
    
    onSettled: (_, __, { id }) => {
      queryClient.invalidateQueries({ queryKey: taskKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}

/**
 * Mutation to delete a task with optimistic update
 */
export function useDeleteTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => taskApi.delete(id),
    
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: taskKeys.lists() });
      
      const previousLists = queryClient.getQueriesData<TaskListResponse>({ queryKey: taskKeys.lists() });
      const previousDetail = queryClient.getQueryData<Task>(taskKeys.detail(id));
      
      queryClient.setQueriesData<TaskListResponse>(
        { queryKey: taskKeys.lists() },
        (old) => {
          if (!old || !old.data) return old;
          return {
            ...old,
            data: old.data.filter((task) => task.id !== id),
            total: (old.total ?? 0) - 1,
          };
        }
      );
      
      return { previousLists, previousDetail };
    },
    
    onError: (err, id, context) => {
      if (context?.previousLists) {
        context.previousLists.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
      if (context?.previousDetail) {
        queryClient.setQueryData(taskKeys.detail(id), context.previousDetail);
      }
    },
    
    onSuccess: () => {
      toast.success('Task deleted');
    },
    
    onSettled: (_, __, id) => {
      queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
      queryClient.removeQueries({ queryKey: taskKeys.detail(id) });
    },
  });
}
