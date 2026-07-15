'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { notificationService } from '@/features/notification/services/notification-service';
import type { NotificationPreference } from '@/features/notification/types';

export const notificationKeys = {
  all: ['notifications'] as const,
  list: () => [...notificationKeys.all, 'list'] as const,
  status: () => [...notificationKeys.all, 'status'] as const,
  preference: () => [...notificationKeys.all, 'preference'] as const,
};

export function useNotificationStatus() {
  return useQuery({
    queryKey: notificationKeys.status(),
    queryFn: () => notificationService.status(),
    staleTime: 30_000,
    refetchInterval: 60_000,
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: notificationKeys.list(),
    queryFn: () => notificationService.list({ page: 1, perPage: 10, status: 'all' }),
    enabled,
    staleTime: 15_000,
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useNotificationPreference(enabled: boolean) {
  return useQuery({
    queryKey: notificationKeys.preference(),
    queryFn: () => notificationService.preference(),
    enabled,
    staleTime: 60_000,
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useReplaceNotificationReadState() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ notificationId, isRead }: { notificationId: number; isRead: boolean }) =>
      notificationService.replaceReadState(notificationId, { is_read: isRead }),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: () => refreshNotificationState(queryClient),
  });
}

export function useMarkNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (throughId: number) => notificationService.markReadThrough(throughId),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: () => refreshNotificationState(queryClient),
  });
}

export function useReplaceNotificationPreference() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: NotificationPreference) =>
      notificationService.replacePreference(input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: preference => {
      queryClient.setQueryData(notificationKeys.preference(), preference);
    },
  });
}

function refreshNotificationState(queryClient: ReturnType<typeof useQueryClient>): void {
  queryClient.invalidateQueries({ queryKey: notificationKeys.list() });
  queryClient.invalidateQueries({ queryKey: notificationKeys.status() });
}
