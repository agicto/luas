import 'server-only';

import { createHash } from 'node:crypto';

import type { AuthUser } from '@/features/auth/types';
import type {
  AppSetting,
  OrganizationSetting,
  OrganizationSettingKey,
  Setting,
  SettingMutation,
  UserSetting,
  UserSettingKey,
} from '@/features/setting/types';

const maxMockSettingSubjects = 1_000;

interface MockOverride {
  value: string;
  version: number;
  overridden: boolean;
  updatedAt: string;
}

type MockSettingState = Map<string, MockOverride>;

export type MockSettingMutationResult =
  { ok: true; setting: Setting } | { ok: false; error: 'not_found' | 'version_conflict' };

const userStates = new Map<string, MockSettingState>();
const organizationStates = new Map<number, MockSettingState>();

export const mockSettingStore = {
  publicApp(): AppSetting[] {
    return [
      {
        scope: 'app',
        key: 'branding.display_name',
        kind: 'string',
        visibility: 'public',
        value: 'Luas',
        version: 0,
        source: 'default',
        updated_at: null,
      },
      {
        scope: 'app',
        key: 'localization.locale',
        kind: 'enum',
        visibility: 'public',
        value: 'en-US',
        version: 0,
        source: 'default',
        options: ['en-US', 'zh-Hans'],
        updated_at: null,
      },
    ];
  },

  publicETag(): string {
    const digest = createHash('sha256').update(JSON.stringify(this.publicApp())).digest('hex');
    return `"settings-${digest}"`;
  },

  user(user: AuthUser): UserSetting[] {
    const state = stateFor(userStates, user.id);
    return [
      effectiveUserSetting('localization.locale', state.get('localization.locale')),
      effectiveUserSetting('localization.timezone', state.get('localization.timezone')),
    ];
  },

  organization(organizationId: number): OrganizationSetting[] {
    const state = stateFor(organizationStates, organizationId);
    return [effectiveOrganizationSetting(state.get('localization.locale'))];
  },

  setUser(
    user: AuthUser,
    key: UserSettingKey,
    input: SettingMutation,
    expectedVersion: number
  ): MockSettingMutationResult {
    return setOverride(stateFor(userStates, user.id), key, input.value, expectedVersion, record =>
      effectiveUserSetting(key, record)
    );
  },

  resetUser(
    user: AuthUser,
    key: UserSettingKey,
    expectedVersion: number
  ): MockSettingMutationResult {
    return resetOverride(stateFor(userStates, user.id), key, expectedVersion, record =>
      effectiveUserSetting(key, record)
    );
  },

  setOrganization(
    organizationId: number,
    key: OrganizationSettingKey,
    input: SettingMutation,
    expectedVersion: number
  ): MockSettingMutationResult {
    return setOverride(
      stateFor(organizationStates, organizationId),
      key,
      input.value,
      expectedVersion,
      record => effectiveOrganizationSetting(record)
    );
  },

  resetOrganization(
    organizationId: number,
    key: OrganizationSettingKey,
    expectedVersion: number
  ): MockSettingMutationResult {
    return resetOverride(
      stateFor(organizationStates, organizationId),
      key,
      expectedVersion,
      record => effectiveOrganizationSetting(record)
    );
  },

  reset(): void {
    userStates.clear();
    organizationStates.clear();
  },
};

function stateFor<Key>(states: Map<Key, MockSettingState>, subject: Key): MockSettingState {
  const existing = states.get(subject);
  if (existing) return existing;
  if (states.size >= maxMockSettingSubjects) {
    const oldest = states.keys().next().value as Key | undefined;
    if (oldest !== undefined) states.delete(oldest);
  }
  const state = new Map<string, MockOverride>();
  states.set(subject, state);
  return state;
}

function setOverride(
  state: MockSettingState,
  key: string,
  value: string | number | boolean,
  expectedVersion: number,
  effective: (record: MockOverride | undefined) => Setting
): MockSettingMutationResult {
  const current = state.get(key);
  const currentVersion = current?.version ?? 0;
  if (currentVersion !== expectedVersion) {
    return { ok: false, error: 'version_conflict' };
  }
  if (current?.overridden && current.value === String(value)) {
    return { ok: true, setting: effective(current) };
  }
  const updated: MockOverride = {
    value: String(value),
    version: currentVersion + 1,
    overridden: true,
    updatedAt: new Date().toISOString(),
  };
  state.set(key, updated);
  return { ok: true, setting: effective(updated) };
}

function resetOverride(
  state: MockSettingState,
  key: string,
  expectedVersion: number,
  effective: (record: MockOverride | undefined) => Setting
): MockSettingMutationResult {
  const current = state.get(key);
  const currentVersion = current?.version ?? 0;
  if (currentVersion !== expectedVersion) {
    return { ok: false, error: 'version_conflict' };
  }
  if (!current || !current.overridden) {
    return { ok: true, setting: effective(current) };
  }
  const updated: MockOverride = {
    value: '',
    version: currentVersion + 1,
    overridden: false,
    updatedAt: new Date().toISOString(),
  };
  state.set(key, updated);
  return { ok: true, setting: effective(updated) };
}

function effectiveUserSetting(key: UserSettingKey, record: MockOverride | undefined): UserSetting {
  if (key === 'localization.locale') {
    return {
      scope: 'user',
      key,
      kind: 'enum',
      visibility: 'private',
      value: record?.overridden ? (record.value as 'en-US' | 'zh-Hans') : 'en-US',
      version: record?.version ?? 0,
      source: record?.overridden ? 'override' : 'default',
      options: ['en-US', 'zh-Hans'],
      updated_at: record?.updatedAt ?? null,
    };
  }
  return {
    scope: 'user',
    key,
    kind: 'timezone',
    visibility: 'private',
    value: record?.overridden ? record.value : 'UTC',
    version: record?.version ?? 0,
    source: record?.overridden ? 'override' : 'default',
    updated_at: record?.updatedAt ?? null,
  };
}

function effectiveOrganizationSetting(record: MockOverride | undefined): OrganizationSetting {
  return {
    scope: 'organization',
    key: 'localization.locale',
    kind: 'enum',
    visibility: 'private',
    value: record?.overridden ? (record.value as 'en-US' | 'zh-Hans') : 'en-US',
    version: record?.version ?? 0,
    source: record?.overridden ? 'override' : 'default',
    options: ['en-US', 'zh-Hans'],
    updated_at: record?.updatedAt ?? null,
  };
}
