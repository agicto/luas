'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { userApi } from '@/services';
import type { CreateUserDto, UpdateUserDto, User, UserListParams, UserListResponse } from '@/services';
import { eventBus } from '@/utils';
import { toast } from 'sonner';

// =============== Query Keys Factory ===============

/**
 * Query key factory - prevents typos and enables type-safe invalidation
 * 
 * Structure:
 * - all: ['users'] - base key for all user-related queries
 * - lists: ['users', 'list'] - all paginated list queries
 * - list: ['users', 'list', params] - specific list with params
 * - all-list: ['users', 'all'] - non-paginated full list
 * - details: ['users', 'detail'] - all detail queries
 * - detail: ['users', 'detail', id] - specific user detail
 */
export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (params?: UserListParams) => [...userKeys.lists(), params] as const,
  allList: () => [...userKeys.all, 'all'] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

// =============== Query Hooks ===============

/**
 * Fetch paginated users list
 * 
 * @example
 * ```tsx
 * const { data, isLoading } = useUsers({ page: 1, limit: 10 });
 * // data.data - User[]
 * // data.total - total count
 * // data.page - current page
 * // data.limit - page size
 * ```
 */
export function useUsers(params?: UserListParams) {
  return useQuery({
    queryKey: userKeys.list(params),
    queryFn: () => userApi.list(params),
  });
}

/**
 * Fetch all users without pagination
 * Useful for dropdowns, selects, or when full list is needed
 * 
 * @example
 * ```tsx
 * const { data: users, isLoading } = useAllUsers();
 * // users - User[] (all users)
 * ```
 */
export function useAllUsers() {
  return useQuery({
    queryKey: userKeys.allList(),
    queryFn: async () => {
      // Fetch with a high limit to get all users
      const response = await userApi.list({ limit: 9999 });
      return response.data;
    },
  });
}

/**
 * Fetch a single user by ID
 * 
 * @example
 * ```tsx
 * const { data: user, isLoading } = useUser('user-id');
 * ```
 */
export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => userApi.get(id),
    enabled: !!id,
  });
}

// =============== Mutation Hooks with Optimistic Updates ===============

/**
 * Create a user with optimistic update
 * 
 * Features:
 * - Optimistically adds user to list cache
 * - Shows toast on success/error
 * - Invalidates list queries on success to refetch with server-assigned ID
 * 
 * @example
 * ```tsx
 * const { mutate: createUser, isPending } = useCreateUser();
 * createUser({ name: 'John', email: 'john@example.com' });
 * ```
 */
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserDto) => userApi.create(data),
    
    onMutate: async (newUserData) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: userKeys.lists() });
      await queryClient.cancelQueries({ queryKey: userKeys.allList() });
      
      // Snapshot previous values for rollback
      const previousLists = queryClient.getQueriesData<UserListResponse>({ queryKey: userKeys.lists() });
      const previousAllList = queryClient.getQueryData<User[]>(userKeys.allList());
      
      // Create optimistic user with temporary ID
      const optimisticUser: User = {
        id: `temp-${Date.now()}`,
        name: newUserData.name,
        email: newUserData.email,
        createdAt: new Date().toISOString(),
      };
      
      // Optimistically update paginated lists
      queryClient.setQueriesData<UserListResponse>(
        { queryKey: userKeys.lists() },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            data: [optimisticUser, ...old.data],
            total: old.total + 1,
          };
        }
      );
      
      // Optimistically update all users list
      queryClient.setQueryData<User[]>(
        userKeys.allList(),
        (old) => old ? [optimisticUser, ...old] : [optimisticUser]
      );
      
      return { previousLists, previousAllList };
    },
    
    onError: (error, _variables, context) => {
      // Rollback to previous state on error
      if (context?.previousLists) {
        context.previousLists.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
      if (context?.previousAllList) {
        queryClient.setQueryData(userKeys.allList(), context.previousAllList);
      }
      
      toast.error('创建用户失败', {
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    },
    
    onSuccess: (data: User) => {
      toast.success('用户创建成功', {
        description: `用户 ${data.name} 已创建`,
      });
      eventBus.publish('user:created', data);
    },
    
    onSettled: () => {
      // Invalidate and refetch to get server-assigned ID
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
      queryClient.invalidateQueries({ queryKey: userKeys.allList() });
    },
  });
}

/**
 * Update a user with optimistic update
 * 
 * Features:
 * - Optimistically updates user in list and detail caches
 * - Rolls back on error
 * - Shows toast on success/error
 * - Invalidates queries to ensure consistency
 * 
 * @example
 * ```tsx
 * const { mutate: updateUser, isPending } = useUpdateUser();
 * updateUser({ id: 'user-id', data: { name: 'New Name' } });
 * ```
 */
export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserDto }) =>
      userApi.update(id, data),
    
    onMutate: async ({ id, data }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: userKeys.detail(id) });
      await queryClient.cancelQueries({ queryKey: userKeys.lists() });
      await queryClient.cancelQueries({ queryKey: userKeys.allList() });
      
      // Snapshot previous values
      const previousDetail = queryClient.getQueryData<User>(userKeys.detail(id));
      const previousLists = queryClient.getQueriesData<UserListResponse>({ queryKey: userKeys.lists() });
      const previousAllList = queryClient.getQueryData<User[]>(userKeys.allList());
      
      // Optimistically update detail cache
      if (previousDetail) {
        queryClient.setQueryData<User>(userKeys.detail(id), {
          ...previousDetail,
          ...data,
        });
      }
      
      // Optimistically update paginated lists
      queryClient.setQueriesData<UserListResponse>(
        { queryKey: userKeys.lists() },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            data: old.data.map((user) =>
              user.id === id ? { ...user, ...data } : user
            ),
          };
        }
      );
      
      // Optimistically update all users list
      queryClient.setQueryData<User[]>(
        userKeys.allList(),
        (old) => old?.map((user) => user.id === id ? { ...user, ...data } : user)
      );
      
      return { previousDetail, previousLists, previousAllList };
    },
    
    onError: (error, { id }, context) => {
      // Rollback all caches on error
      if (context?.previousDetail) {
        queryClient.setQueryData(userKeys.detail(id), context.previousDetail);
      }
      if (context?.previousLists) {
        context.previousLists.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
      if (context?.previousAllList) {
        queryClient.setQueryData(userKeys.allList(), context.previousAllList);
      }
      
      toast.error('更新用户失败', {
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    },
    
    onSuccess: (updatedUser: User) => {
      toast.success('用户更新成功', {
        description: `用户 ${updatedUser.name} 已更新`,
      });
      eventBus.publish('user:updated', updatedUser);
    },
    
    onSettled: (_data, _error, { id }) => {
      // Invalidate to ensure consistency with server
      queryClient.invalidateQueries({ queryKey: userKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
      queryClient.invalidateQueries({ queryKey: userKeys.allList() });
    },
  });
}

/**
 * Delete a user with optimistic update
 * 
 * Features:
 * - Optimistically removes user from list cache
 * - Rolls back on error
 * - Shows toast on success/error
 * - Removes detail cache and invalidates lists on success
 * 
 * @example
 * ```tsx
 * const { mutate: deleteUser, isPending } = useDeleteUser();
 * deleteUser('user-id');
 * ```
 */
export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => userApi.delete(id),
    
    onMutate: async (id) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: userKeys.lists() });
      await queryClient.cancelQueries({ queryKey: userKeys.allList() });
      
      // Snapshot previous values
      const previousLists = queryClient.getQueriesData<UserListResponse>({ queryKey: userKeys.lists() });
      const previousAllList = queryClient.getQueryData<User[]>(userKeys.allList());
      const previousDetail = queryClient.getQueryData<User>(userKeys.detail(id));
      
      // Optimistically remove from paginated lists
      queryClient.setQueriesData<UserListResponse>(
        { queryKey: userKeys.lists() },
        (old) => {
          if (!old) return old;
          return {
            ...old,
            data: old.data.filter((user) => user.id !== id),
            total: old.total - 1,
          };
        }
      );
      
      // Optimistically remove from all users list
      queryClient.setQueryData<User[]>(
        userKeys.allList(),
        (old) => old?.filter((user) => user.id !== id)
      );
      
      return { previousLists, previousAllList, previousDetail };
    },
    
    onError: (error, id, context) => {
      // Rollback all caches on error
      if (context?.previousLists) {
        context.previousLists.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
      if (context?.previousAllList) {
        queryClient.setQueryData(userKeys.allList(), context.previousAllList);
      }
      if (context?.previousDetail) {
        queryClient.setQueryData(userKeys.detail(id), context.previousDetail);
      }
      
      toast.error('删除用户失败', {
        description: error instanceof Error ? error.message : '请稍后重试',
      });
    },
    
    onSuccess: (_, id) => {
      toast.success('用户删除成功');
      eventBus.publish('user:deleted', id);
    },
    
    onSettled: (_, _error, id) => {
      // Invalidate lists and remove detail query
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
      queryClient.invalidateQueries({ queryKey: userKeys.allList() });
      queryClient.removeQueries({ queryKey: userKeys.detail(id) });
    },
  });
}
