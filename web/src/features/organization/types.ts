import type { infer as Infer } from 'zod/mini';

import type {
  createOrganizationSchema,
  organizationContextSchema,
  organizationRoleSchema,
  organizationSchema,
  paginationLinksSchema,
  paginationMetaSchema,
  updateOrganizationSchema,
} from './schemas';

export type OrganizationRole = Infer<typeof organizationRoleSchema>;
export type Organization = Infer<typeof organizationSchema>;
export type OrganizationContext = Infer<typeof organizationContextSchema>;
export type PaginationMeta = Infer<typeof paginationMetaSchema>;
export type PaginationLinks = Infer<typeof paginationLinksSchema>;
export type CreateOrganizationInput = Infer<typeof createOrganizationSchema>;
export type UpdateOrganizationInput = Infer<typeof updateOrganizationSchema>;

export interface OrganizationPage {
  items: Organization[];
  meta: PaginationMeta;
  links: PaginationLinks;
}
