import { z } from 'zod';

export const auditEntrySchema = z.object({
  id: z.number().int().positive(),
  user_id: z.number().int().positive().optional(),
  actor_type: z.string(),
  action: z.string(),
  resource: z.string(),
  target_type: z.string().optional(),
  target_id: z.string().optional(),
  result: z.string().optional(),
  method: z.string(),
  path: z.string(),
  status_code: z.number().int(),
  request_id: z.string().optional(),
  created_at: z.string(),
});

export const auditPageSchema = z.object({
  code: z.literal(0),
  data: z.array(auditEntrySchema),
  meta: z.object({
    current_page: z.number().int().min(1),
    last_page: z.number().int().min(1),
    per_page: z.number().int().min(1),
    total: z.number().int().min(0),
  }),
});

export type AuditEntry = z.infer<typeof auditEntrySchema>;
export type AuditPage = z.infer<typeof auditPageSchema>;

/** Filters kept in the URL. `from` and `to` are local calendar days (YYYY-MM-DD); `to` is inclusive. */
export interface AuditSearch {
  action?: string | undefined;
  user_id?: number | undefined;
  from?: string | undefined;
  to?: string | undefined;
  page?: number | undefined;
}
