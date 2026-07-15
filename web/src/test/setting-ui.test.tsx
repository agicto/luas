import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { OrganizationSettingPanel } from '@/features/setting/components/organization-setting-panel';
import { UserSettingPanel } from '@/features/setting/components/user-setting-panel';
import { messages } from '@/i18n/modules';

const toast = vi.hoisted(() => ({ success: vi.fn() }));
const state = vi.hoisted(() => ({
  userQuery: {
    data: undefined as unknown,
    isPending: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
  userSet: { isPending: false, error: null as unknown, mutateAsync: vi.fn() },
  userReset: { isPending: false, error: null as unknown, mutate: vi.fn() },
  organizationQuery: {
    data: undefined as unknown,
    isPending: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
  organizationSet: { isPending: false, error: null as unknown, mutate: vi.fn() },
  organizationReset: { isPending: false, error: null as unknown, mutate: vi.fn() },
}));

vi.mock('sonner', () => ({ toast }));
vi.mock('@/features/setting/hooks/use-settings', () => ({
  useUserSettings: () => state.userQuery,
  useSetUserSetting: () => state.userSet,
  useResetUserSetting: () => state.userReset,
  useOrganizationSettings: () => state.organizationQuery,
  useSetOrganizationSetting: () => state.organizationSet,
  useResetOrganizationSetting: () => state.organizationReset,
}));

describe('typed setting preferences', () => {
  beforeEach(() => {
    state.userQuery.data = userSettings();
    state.userQuery.isPending = false;
    state.userQuery.error = null;
    state.userQuery.refetch.mockReset();
    state.userSet.isPending = false;
    state.userSet.error = null;
    state.userSet.mutateAsync.mockReset();
    state.userSet.mutateAsync.mockResolvedValue(undefined);
    state.userReset.isPending = false;
    state.userReset.error = null;
    state.userReset.mutate.mockReset();
    state.organizationQuery.data = [organizationSetting()];
    state.organizationQuery.isPending = false;
    state.organizationQuery.error = null;
    state.organizationQuery.refetch.mockReset();
    state.organizationSet.isPending = false;
    state.organizationSet.error = null;
    state.organizationSet.mutate.mockReset();
    state.organizationReset.isPending = false;
    state.organizationReset.error = null;
    state.organizationReset.mutate.mockReset();
    toast.success.mockReset();
  });

  it('saves a validated IANA timezone without rendering legacy fake controls', async () => {
    renderWithMessages(<UserSettingPanel />);

    fireEvent.change(screen.getByLabelText('Time zone'), {
      target: { value: 'Europe/Dublin' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save preferences' }));

    await waitFor(() =>
      expect(state.userSet.mutateAsync).toHaveBeenCalledWith({
        key: 'localization.timezone',
        input: { value: 'Europe/Dublin' },
        expectedVersion: 0,
      })
    );
    expect(toast.success).toHaveBeenCalledWith('Preferences saved');
    expect(screen.queryByText(/Company URL|Support email|Two-factor|SMS|Push/i)).toBeNull();
  });

  it('resets an override with its current version', () => {
    state.userQuery.data = [
      { ...userSettings()[0], value: 'zh-Hans', version: 3, source: 'override' },
      userSettings()[1],
    ];
    renderWithMessages(<UserSettingPanel />);

    fireEvent.click(screen.getByRole('button', { name: 'Reset locale to default' }));

    expect(state.userReset.mutate).toHaveBeenCalledWith(
      { key: 'localization.locale', expectedVersion: 3 },
      expect.objectContaining({ onSuccess: expect.any(Function) })
    );
  });

  it('keeps organization preferences read-only for ordinary members', () => {
    renderWithMessages(<OrganizationSettingPanel organizationId={7} canManage={false} />);

    expect(screen.getByRole('combobox', { name: 'Locale' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Save preferences' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Reset locale to default' })).toBeNull();
  });
});

function userSettings() {
  return [
    {
      scope: 'user' as const,
      key: 'localization.locale' as const,
      kind: 'enum' as const,
      visibility: 'private' as const,
      value: 'en-US' as const,
      version: 0,
      source: 'default' as const,
      options: ['en-US', 'zh-Hans'] as const,
      updated_at: null,
    },
    {
      scope: 'user' as const,
      key: 'localization.timezone' as const,
      kind: 'timezone' as const,
      visibility: 'private' as const,
      value: 'UTC',
      version: 0,
      source: 'default' as const,
      updated_at: null,
    },
  ];
}

function organizationSetting() {
  return {
    scope: 'organization' as const,
    key: 'localization.locale' as const,
    kind: 'enum' as const,
    visibility: 'private' as const,
    value: 'en-US' as const,
    version: 0,
    source: 'default' as const,
    options: ['en-US', 'zh-Hans'] as const,
    updated_at: null,
  };
}

function renderWithMessages(children: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}
