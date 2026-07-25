import common, { type CommonMessages } from './common/en-US';
import auth, { type AuthMessages } from './auth/en-US';
import nav, { type NavMessages } from './nav/en-US';
import site, { type SiteMessages } from './site/en-US';
import console, { type ConsoleMessages } from './console/en-US';
import organization, { type OrganizationMessages } from './organization/en-US';
import permission, { type PermissionMessages } from './permission/en-US';
import notification, { type NotificationMessages } from './notification/en-US';
import asset, { type AssetMessages } from './asset/en-US';
import setting, { type SettingMessages } from './setting/en-US';
import usage, { type UsageMessages } from './usage/en-US';
import webhook, { type WebhookMessages } from './webhook/en-US';
import settings, { type SettingsMessages } from './settings/en-US';
import errors, { type ErrorsMessages } from './errors/en-US';
import metadata, { type MetadataMessages } from './metadata/en-US';
import test, { type TestMessages } from './test/en-US';
import type { LocaleMessageVariableParity } from '../locale-message-shape';
import type { Locale } from '../locales';

/**
 * Canonical source messages used for IDE completion and locale parity checks.
 */
export const messages = {
  common,
  auth,
  nav,
  site,
  console,
  organization,
  permission,
  notification,
  asset,
  setting,
  usage,
  webhook,
  settings,
  errors,
  metadata,
  test,
} as const;

export type Messages = typeof messages;

/**
 * Canonical message schema. Source-locale literals retain ICU variable names.
 */
export interface MessageSchema {
  common: CommonMessages;
  auth: AuthMessages;
  nav: NavMessages;
  site: SiteMessages;
  console: ConsoleMessages;
  organization: OrganizationMessages;
  permission: PermissionMessages;
  notification: NotificationMessages;
  asset: AssetMessages;
  setting: SettingMessages;
  usage: UsageMessages;
  webhook: WebhookMessages;
  settings: SettingsMessages;
  errors: ErrorsMessages;
  metadata: MetadataMessages;
  test: TestMessages;
}

interface ZhHansMessageSchema {
  common: typeof import('./common/zh-Hans').default;
  auth: typeof import('./auth/zh-Hans').default;
  nav: typeof import('./nav/zh-Hans').default;
  site: typeof import('./site/zh-Hans').default;
  console: typeof import('./console/zh-Hans').default;
  organization: typeof import('./organization/zh-Hans').default;
  permission: typeof import('./permission/zh-Hans').default;
  notification: typeof import('./notification/zh-Hans').default;
  asset: typeof import('./asset/zh-Hans').default;
  setting: typeof import('./setting/zh-Hans').default;
  usage: typeof import('./usage/zh-Hans').default;
  webhook: typeof import('./webhook/zh-Hans').default;
  settings: typeof import('./settings/zh-Hans').default;
  errors: typeof import('./errors/zh-Hans').default;
  metadata: typeof import('./metadata/zh-Hans').default;
  test: typeof import('./test/zh-Hans').default;
}

type Assert<T extends true> = T;

interface LocaleMessageSchemas {
  'en-US': Messages;
  'zh-Hans': ZhHansMessageSchema;
}

type ConfiguredLocalesMatchSchemas = [Locale] extends [keyof LocaleMessageSchemas]
  ? [keyof LocaleMessageSchemas] extends [Locale]
    ? true
    : false
  : false;

type EveryLocaleMatchesBaseVariables = false extends {
  [ConfiguredLocale in keyof LocaleMessageSchemas]: LocaleMessageVariableParity<
    MessageSchema,
    LocaleMessageSchemas[ConfiguredLocale]
  >;
}[keyof LocaleMessageSchemas]
  ? false
  : true;

/** Compile-time guard: configured locales and typed locale schemas stay aligned. */
export type LocaleMessageSchemaCoverageCheck = Assert<ConfiguredLocalesMatchSchemas>;

/**
 * Compile-time guard: every locale must preserve the source locale's ICU names.
 */
export type LocaleMessageVariableParityCheck = Assert<EveryLocaleMatchesBaseVariables>;
