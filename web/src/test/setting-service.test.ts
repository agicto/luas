import { describe, expect, it } from 'vitest';

import {
  parseAppSettings,
  parseOrganizationSettings,
  parseUserSettings,
} from '@/features/setting/services/setting-service';

describe('setting service contract parsing', () => {
  it('accepts the exact shipped app, organization, and user definitions', () => {
    expect(
      parseAppSettings([
        setting('app', 'branding.display_name', 'string', 'public', 'Luas'),
        setting('app', 'localization.locale', 'enum', 'public', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
      ])
    ).toHaveLength(2);
    expect(
      parseOrganizationSettings([
        setting('organization', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
      ])
    ).toHaveLength(1);
    expect(
      parseUserSettings([
        setting('user', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
        setting('user', 'localization.timezone', 'timezone', 'private', 'UTC'),
      ])
    ).toHaveLength(2);
  });

  it('fails closed for unknown, missing, duplicate, and malformed definitions', () => {
    expect(() =>
      parseUserSettings([
        setting('user', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
      ])
    ).toThrow('invalid response');

    expect(() =>
      parseUserSettings([
        setting('user', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
        setting('user', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
      ])
    ).toThrow('invalid response');

    expect(() =>
      parseAppSettings([
        setting('app', 'branding.display_name', 'string', 'public', 'Luas'),
        setting('app', 'localization.locale', 'enum', 'public', 'en-US', {
          options: ['zh-Hans', 'en-US'],
        }),
      ])
    ).toThrow('invalid response');

    expect(() =>
      parseUserSettings([
        setting('user', 'localization.locale', 'enum', 'private', 'en-US', {
          options: ['en-US', 'zh-Hans'],
        }),
        setting('user', 'localization.timezone', 'timezone', 'private', 'Not/AZone'),
      ])
    ).toThrow('invalid response');
  });
});

function setting(
  scope: string,
  key: string,
  kind: string,
  visibility: string,
  value: unknown,
  extra: Record<string, unknown> = {}
) {
  return {
    scope,
    key,
    kind,
    visibility,
    value,
    version: 0,
    source: 'default',
    updated_at: null,
    ...extra,
  };
}
