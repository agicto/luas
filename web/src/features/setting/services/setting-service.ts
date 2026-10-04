import type { z } from 'zod/mini';
import {
  assertContract,
  type Accepts,
  type ContractData,
  type ContractRequest,
  type Sends,
} from '@/http/contract';
import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  settingSchema,
  settingWireListSchema,
  settingWireSchema,
} from '@/features/setting/schemas';
import type {
  AppSetting,
  OrganizationSetting,
  OrganizationSettingKey,
  Setting,
  SettingMutation,
  UserSetting,
  UserSettingKey,
} from '@/features/setting/types';

const expectedAppSettings = ['app:branding.display_name', 'app:localization.locale'] as const;
const expectedUserSettings = ['user:localization.locale', 'user:localization.timezone'] as const;
const expectedOrganizationSettings = ['organization:localization.locale'] as const;

export const settingService = {
  async publicApp(): Promise<AppSetting[]> {
    return parseAppSettings(await request.get<unknown>('/settings/public'));
  },

  async user(): Promise<UserSetting[]> {
    return parseUserSettings(await request.get<unknown>('/settings/user'));
  },

  async setUser(
    key: UserSettingKey,
    input: SettingMutation,
    expectedVersion: number
  ): Promise<UserSetting> {
    const value = await request.patch<unknown, SettingMutation>(`/settings/user/${key}`, input, {
      headers: { 'If-Match': versionETag(expectedVersion) },
    });
    return parseExpectedSetting(value, 'user', key) as UserSetting;
  },

  async resetUser(key: UserSettingKey, expectedVersion: number): Promise<void> {
    await request.delete<void>(`/settings/user/${key}`, {
      headers: { 'If-Match': versionETag(expectedVersion) },
    });
  },

  async organization(organizationId: number): Promise<OrganizationSetting[]> {
    return parseOrganizationSettings(
      await request.get<unknown>('/organization-settings', {
        headers: { 'Organization-Id': String(organizationId) },
      })
    );
  },

  async setOrganization(
    organizationId: number,
    key: OrganizationSettingKey,
    input: SettingMutation,
    expectedVersion: number
  ): Promise<OrganizationSetting> {
    const value = await request.patch<unknown, SettingMutation>(
      `/organization-settings/${key}`,
      input,
      {
        headers: {
          'Organization-Id': String(organizationId),
          'If-Match': versionETag(expectedVersion),
        },
      }
    );
    return parseExpectedSetting(value, 'organization', key) as OrganizationSetting;
  },

  async resetOrganization(
    organizationId: number,
    key: OrganizationSettingKey,
    expectedVersion: number
  ): Promise<void> {
    await request.delete<void>(`/organization-settings/${key}`, {
      headers: {
        'Organization-Id': String(organizationId),
        'If-Match': versionETag(expectedVersion),
      },
    });
  },
};

export function parseAppSettings(value: unknown): AppSetting[] {
  return parseExpectedList(value, expectedAppSettings) as AppSetting[];
}

export function parseUserSettings(value: unknown): UserSetting[] {
  return parseExpectedList(value, expectedUserSettings) as UserSetting[];
}

export function parseOrganizationSettings(value: unknown): OrganizationSetting[] {
  return parseExpectedList(value, expectedOrganizationSettings) as OrganizationSetting[];
}

export function parseExpectedSetting(
  value: unknown,
  scope: Setting['scope'],
  key: Setting['key']
): Setting {
  const wire = settingWireSchema.safeParse(value);
  if (!wire.success || wire.data.scope !== scope || wire.data.key !== key) {
    throw invalidResponse();
  }
  const parsed = settingSchema.safeParse(wire.data);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

/**
 * Returns the expected definitions in catalog order. Each must be present and valid; definitions
 * the server adds beyond them are ignored so an API with a larger catalog does not break this page.
 */
function parseExpectedList(value: unknown, expected: readonly string[]): Setting[] {
  const wire = settingWireListSchema.safeParse(value);
  if (!wire.success) throw invalidResponse();
  return expected.map(identity => {
    const item = wire.data.find(entry => `${entry.scope}:${entry.key}` === identity);
    const parsed = settingSchema.safeParse(item);
    if (!parsed.success) throw invalidResponse();
    return parsed.data;
  });
}

function versionETag(version: number): string {
  if (!Number.isSafeInteger(version) || version < 0) {
    throw new ApiError('Setting version is invalid', ClientErrorCode.INVALID_RESPONSE);
  }
  return `"setting-v${version}"`;
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Setting service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}

assertContract<
  Accepts<z.input<typeof settingWireListSchema>, ContractData<'listPublicSettings', 200>>
>();
assertContract<
  Accepts<z.input<typeof settingWireListSchema>, ContractData<'listUserSettings', 200>>
>();
assertContract<
  Accepts<z.input<typeof settingWireSchema>, ContractData<'updateUserSetting', 200>>
>();
assertContract<Sends<SettingMutation, ContractRequest<'updateUserSetting'>>>();
assertContract<
  Accepts<z.input<typeof settingWireListSchema>, ContractData<'listOrganizationSettings', 200>>
>();
assertContract<
  Accepts<z.input<typeof settingWireSchema>, ContractData<'updateOrganizationSetting', 200>>
>();
assertContract<Sends<SettingMutation, ContractRequest<'updateOrganizationSetting'>>>();
