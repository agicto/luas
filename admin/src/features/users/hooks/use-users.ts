import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { usersService } from '@/features/users/services/users-service';
import type { UserAction, UserSearch } from '@/features/users/types';

export const usersQueryKey = ['operator', 'users'] as const;

export function useUsers(search: UserSearch) {
  return useQuery({
    queryKey: [...usersQueryKey, search],
    queryFn: ({ signal }) => usersService.list(search, signal),
    placeholderData: keepPreviousData,
  });
}

export function useUserAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ action, id }: { action: UserAction; id: number }) => {
      if (action === 'disable') {
        await usersService.disable(id);
      } else if (action === 'enable') {
        await usersService.enable(id);
      } else {
        await usersService.revokeSessions(id);
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: usersQueryKey }),
  });
}
