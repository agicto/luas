'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { settingService } from '@/features/setting/services/setting-service';
import type {
  OrganizationSetting,
  OrganizationSettingKey,
  SettingMutation,
  UserSetting,
  UserSettingKey,
} from '@/features/setting/types';

export const settingKeys = {
  all: ['settings'] as const,
  user: () => [...settingKeys.all, 'user'] as const,
  organization: (organizationId: number) =>
    [...settingKeys.all, 'organization', organizationId] as const,
};

export function useUserSettings() {
  return useQuery({
    queryKey: settingKeys.user(),
    queryFn: () => settingService.user(),
    staleTime: 30_000,
  });
}

export function useSetUserSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      key,
      input,
      expectedVersion,
    }: {
      key: UserSettingKey;
      input: SettingMutation;
      expectedVersion: number;
    }) => settingService.setUser(key, input, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: setting => {
      queryClient.setQueryData<UserSetting[]>(settingKeys.user(), current =>
        replaceSetting(current, setting)
      );
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: settingKeys.user() }),
  });
}

export function useResetUserSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ key, expectedVersion }: { key: UserSettingKey; expectedVersion: number }) =>
      settingService.resetUser(key, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () => queryClient.invalidateQueries({ queryKey: settingKeys.user() }),
  });
}

export function useOrganizationSettings(organizationId: number) {
  return useQuery({
    queryKey: settingKeys.organization(organizationId),
    queryFn: () => settingService.organization(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 30_000,
  });
}

export function useSetOrganizationSetting(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      key,
      input,
      expectedVersion,
    }: {
      key: OrganizationSettingKey;
      input: SettingMutation;
      expectedVersion: number;
    }) => settingService.setOrganization(organizationId, key, input, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: setting => {
      queryClient.setQueryData<OrganizationSetting[]>(
        settingKeys.organization(organizationId),
        current => replaceSetting(current, setting)
      );
    },
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: settingKeys.organization(organizationId),
      }),
  });
}

export function useResetOrganizationSetting(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      key,
      expectedVersion,
    }: {
      key: OrganizationSettingKey;
      expectedVersion: number;
    }) => settingService.resetOrganization(organizationId, key, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: settingKeys.organization(organizationId),
      }),
  });
}

function replaceSetting<T extends { key: string }>(
  current: T[] | undefined,
  setting: T
): T[] | undefined {
  return current?.map(item => (item.key === setting.key ? setting : item));
}
