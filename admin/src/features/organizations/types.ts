import { z } from 'zod';
import { pageSchema } from '@/http/pagination';

export const operatorOrganizationSchema = z.object({
  id: z.number().int().positive(),
  name: z.string(),
  slug: z.string(),
  created_by: z.number().int().positive(),
  member_count: z.number().int().min(0),
  created_at: z.string(),
  updated_at: z.string(),
});

export const organizationRoles = ['owner', 'admin', 'member'] as const;

export const operatorMemberSchema = z.object({
  id: z.number().int().positive(),
  user_id: z.number().int().positive(),
  username: z.string(),
  nickname: z.string(),
  email: z.string(),
  role: z.enum(organizationRoles),
  joined_at: z.string(),
});

export const operatorOrganizationPageSchema = pageSchema(operatorOrganizationSchema);
export const operatorMemberPageSchema = pageSchema(operatorMemberSchema);

export type OperatorOrganization = z.infer<typeof operatorOrganizationSchema>;
export type OperatorOrganizationPage = z.infer<typeof operatorOrganizationPageSchema>;
export type OperatorMember = z.infer<typeof operatorMemberSchema>;
export type OperatorMemberPage = z.infer<typeof operatorMemberPageSchema>;

export interface OrganizationSearch {
  q?: string | undefined;
  page?: number | undefined;
}
