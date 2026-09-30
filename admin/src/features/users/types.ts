import { z } from 'zod';

export const managedUserSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.string(),
  nickname: z.string(),
  status: z.enum(['active', 'disabled']),
  is_operator: z.boolean(),
  created_at: z.string(),
  last_login: z.string().nullable(),
});

export const managedUserPageSchema = z.object({
  code: z.literal(0),
  data: z.array(managedUserSchema),
  meta: z.object({
    current_page: z.number().int().min(1),
    last_page: z.number().int().min(1),
    per_page: z.number().int().min(1),
    total: z.number().int().min(0),
  }),
});

export type ManagedUser = z.infer<typeof managedUserSchema>;
export type ManagedUserPage = z.infer<typeof managedUserPageSchema>;

export const userStatusFilters = ['all', 'active', 'disabled'] as const;
export type UserStatusFilter = (typeof userStatusFilters)[number];

export interface UserSearch {
  q?: string | undefined;
  status?: UserStatusFilter | undefined;
  page?: number | undefined;
}

export type UserAction = 'disable' | 'enable' | 'revokeSessions';
