import type { infer as Infer } from 'zod/mini';

import type {
  localeSettingMutationSchema,
  organizationSettingRouteKeySchema,
  settingSchema,
  timezoneSettingMutationSchema,
  userSettingRouteKeySchema,
} from './schemas';

export type Setting = Infer<typeof settingSchema>;
export type LocaleSettingMutation = Infer<typeof localeSettingMutationSchema>;
export type TimezoneSettingMutation = Infer<typeof timezoneSettingMutationSchema>;
export type UserSettingKey = Infer<typeof userSettingRouteKeySchema>;
export type OrganizationSettingKey = Infer<typeof organizationSettingRouteKeySchema>;
export type SettingMutation = LocaleSettingMutation | TimezoneSettingMutation;

export type UserSetting = Extract<Setting, { scope: 'user' }>;
export type OrganizationSetting = Extract<Setting, { scope: 'organization' }>;
export type AppSetting = Extract<Setting, { scope: 'app' }>;
