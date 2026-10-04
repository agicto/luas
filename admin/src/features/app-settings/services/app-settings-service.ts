import type { z } from 'zod';
import { assertContract, type Accepts, type ContractData, type ContractRequest, type Sends } from '@/http/contract';
import { http } from '@/http/client';
import {
  appSettingListSchema,
  appSettingSchema,
  type AppSetting,
  type AppSettingValue,
} from '@/features/app-settings/types';

const ifMatch = (version: number) => ({ 'If-Match': `"setting-v${version}"` });

export const appSettingsService = {
  list(signal?: AbortSignal): Promise<AppSetting[]> {
    return http.get('/v1/operator/settings', {
      schema: appSettingListSchema,
      ...(signal ? { signal } : {}),
    });
  },
  set(key: string, value: AppSettingValue, version: number): Promise<AppSetting> {
    return http.patch(
      `/v1/operator/settings/${encodeURIComponent(key)}`,
      { value },
      { headers: ifMatch(version), schema: appSettingSchema },
    );
  },
  reset(key: string, version: number): Promise<void> {
    return http.delete(`/v1/operator/settings/${encodeURIComponent(key)}`, {
      headers: ifMatch(version),
      responseMode: 'json',
    });
  },
};

// Fails type-check when contracts/openapi.yaml and these schemas disagree.
assertContract<Accepts<z.input<typeof appSettingListSchema>, ContractData<'listOperatorAppSettings', 200>>>();
assertContract<Accepts<z.input<typeof appSettingSchema>, ContractData<'setOperatorAppSetting', 200>>>();
assertContract<Sends<{ value: AppSettingValue }, ContractRequest<'setOperatorAppSetting'>>>();
