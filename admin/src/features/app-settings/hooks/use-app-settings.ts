import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { appSettingsService } from '@/features/app-settings/services/app-settings-service';
import type { AppSettingValue } from '@/features/app-settings/types';

export const appSettingsQueryKey = ['operator', 'app-settings'] as const;

export function useAppSettings() {
  return useQuery({
    queryKey: appSettingsQueryKey,
    queryFn: ({ signal }) => appSettingsService.list(signal),
  });
}

type SettingChange =
  | { kind: 'set'; key: string; value: AppSettingValue; version: number }
  | { kind: 'reset'; key: string; version: number };

export function useChangeAppSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (change: SettingChange) => {
      if (change.kind === 'set') {
        await appSettingsService.set(change.key, change.value, change.version);
      } else {
        await appSettingsService.reset(change.key, change.version);
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: appSettingsQueryKey }),
  });
}
