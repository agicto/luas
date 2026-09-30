import { z } from 'zod';

export const appSettingSchema = z.object({
  scope: z.literal('app'),
  key: z.string(),
  kind: z.enum(['string', 'boolean', 'integer', 'enum', 'timezone']),
  visibility: z.string(),
  value: z.union([z.string(), z.number(), z.boolean()]),
  version: z.number().int().min(0),
  source: z.enum(['default', 'override']),
  options: z.array(z.string()).optional(),
  updated_at: z.string().nullable(),
});

export const appSettingListSchema = z.array(appSettingSchema);

export type AppSetting = z.infer<typeof appSettingSchema>;
export type AppSettingValue = AppSetting['value'];
