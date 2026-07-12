import common from './common/en-US';
import type { CommonMessages } from './common/zh-Hans';
import auth from './auth/en-US';
import type { AuthMessages } from './auth/zh-Hans';
import nav from './nav/en-US';
import type { NavMessages } from './nav/zh-Hans';
import site from './site/en-US';
import type { SiteMessages } from './site/zh-Hans';
import console from './console/en-US';
import type { ConsoleMessages } from './console/zh-Hans';
import settings from './settings/en-US';
import type { SettingsMessages } from './settings/zh-Hans';
import errors from './errors/en-US';
import type { ErrorsMessages } from './errors/zh-Hans';
import metadata from './metadata/en-US';
import type { MetadataMessages } from './metadata/zh-Hans';
import test from './test/en-US';
import type { TestMessages } from './test/zh-Hans';
import type { LocaleMessageVariableParity } from '../locale-message-shape';
import type { Locale } from '../locales';

/**
 * Static messages type derived from English (en-US) files.
 * This is used for IDE auto-completion and type checking.
 */
export const messages = {
  common,
  auth,
  nav,
  site,
  console,
  settings,
  errors,
  metadata,
  test,
} as const;

export type Messages = typeof messages;

/**
 * Canonical message schema. Literal base-locale strings retain ICU variable names.
 */
export interface MessageSchema {
  common: CommonMessages;
  auth: AuthMessages;
  nav: NavMessages;
  site: SiteMessages;
  console: ConsoleMessages;
  settings: SettingsMessages;
  errors: ErrorsMessages;
  metadata: MetadataMessages;
  test: TestMessages;
}

type Assert<T extends true> = T;

interface LocaleMessageSchemas {
  'zh-Hans': MessageSchema;
  'en-US': Messages;
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
 * Compile-time guard: every locale must preserve the base locale's ICU names.
 */
export type LocaleMessageVariableParityCheck = Assert<EveryLocaleMatchesBaseVariables>;
