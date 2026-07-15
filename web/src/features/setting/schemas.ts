import {
  array,
  int,
  iso,
  literal,
  maxLength,
  maximum,
  nonnegative,
  nullable,
  number,
  refine,
  strictObject,
  string,
  union,
} from 'zod/mini';

const safeVersionSchema = number().check(int(), nonnegative(), maximum(Number.MAX_SAFE_INTEGER));
const sourceSchema = union([literal('default'), literal('override')]);
const updatedAtSchema = nullable(iso.datetime({ offset: true }));
const localeSchema = union([literal('en-US'), literal('zh-Hans')]);
const localeOptionsSchema = array(localeSchema).check(
  maxLength(2),
  refine(options => options.length === 2 && options[0] === 'en-US' && options[1] === 'zh-Hans')
);
const displayNameSchema = string().check(
  refine(value => {
    const length = Array.from(value).length;
    return value === value.trim() && length >= 1 && length <= 80 && !containsControl(value);
  })
);
const timezoneSchema = string().check(
  maxLength(64),
  refine(value => value.length > 0 && value !== 'Local' && isSupportedTimezone(value))
);

const appDisplayNameSettingSchema = strictObject({
  scope: literal('app'),
  key: literal('branding.display_name'),
  kind: literal('string'),
  visibility: literal('public'),
  value: displayNameSchema,
  version: safeVersionSchema,
  source: sourceSchema,
  updated_at: updatedAtSchema,
});

const appLocaleSettingSchema = strictObject({
  scope: literal('app'),
  key: literal('localization.locale'),
  kind: literal('enum'),
  visibility: literal('public'),
  value: localeSchema,
  version: safeVersionSchema,
  source: sourceSchema,
  options: localeOptionsSchema,
  updated_at: updatedAtSchema,
});

const organizationLocaleSettingSchema = strictObject({
  scope: literal('organization'),
  key: literal('localization.locale'),
  kind: literal('enum'),
  visibility: literal('private'),
  value: localeSchema,
  version: safeVersionSchema,
  source: sourceSchema,
  options: localeOptionsSchema,
  updated_at: updatedAtSchema,
});

const userLocaleSettingSchema = strictObject({
  scope: literal('user'),
  key: literal('localization.locale'),
  kind: literal('enum'),
  visibility: literal('private'),
  value: localeSchema,
  version: safeVersionSchema,
  source: sourceSchema,
  options: localeOptionsSchema,
  updated_at: updatedAtSchema,
});

const userTimezoneSettingSchema = strictObject({
  scope: literal('user'),
  key: literal('localization.timezone'),
  kind: literal('timezone'),
  visibility: literal('private'),
  value: timezoneSchema,
  version: safeVersionSchema,
  source: sourceSchema,
  updated_at: updatedAtSchema,
});

export const settingSchema = union([
  appDisplayNameSettingSchema,
  appLocaleSettingSchema,
  organizationLocaleSettingSchema,
  userLocaleSettingSchema,
  userTimezoneSettingSchema,
]);

export const settingListSchema = array(settingSchema).check(
  maxLength(64),
  refine(values => {
    const identities = values.map(value => `${value.scope}:${value.key}`);
    return new Set(identities).size === identities.length;
  })
);

export const localeSettingMutationSchema = strictObject({ value: localeSchema });
export const timezoneSettingMutationSchema = strictObject({ value: timezoneSchema });

export const userSettingRouteKeySchema = union([
  literal('localization.locale'),
  literal('localization.timezone'),
]);
export const organizationSettingRouteKeySchema = literal('localization.locale');

function containsControl(value: string): boolean {
  return /[\u0000-\u001F\u007F]/u.test(value);
}

function isSupportedTimezone(value: string): boolean {
  if (!/^[A-Za-z0-9_+-]+(?:\/[A-Za-z0-9_+-]+)*$/u.test(value)) return false;
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: value }).format();
    return true;
  } catch {
    return false;
  }
}
