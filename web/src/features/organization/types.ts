import type { infer as Infer } from 'zod/mini';

import type {
  acceptOrganizationInvitationSchema,
  createOrganizationInvitationSchema,
  createOrganizationSchema,
  invitationEmailSendStatusSchema,
  organizationContextSchema,
  organizationInvitationRoleSchema,
  organizationInvitationSchema,
  organizationInvitationStatusSchema,
  organizationInvitationCreateResultSchema,
  organizationMemberSchema,
  organizationOwnershipTransferSchema,
  organizationRoleSchema,
  organizationSchema,
  paginationLinksSchema,
  paginationMetaSchema,
  transferOrganizationOwnershipSchema,
  updateOrganizationMemberSchema,
  updateOrganizationSchema,
} from './schemas';

export type OrganizationRole = Infer<typeof organizationRoleSchema>;
export type Organization = Infer<typeof organizationSchema>;
export type OrganizationContext = Infer<typeof organizationContextSchema>;
export type PaginationMeta = Infer<typeof paginationMetaSchema>;
export type PaginationLinks = Infer<typeof paginationLinksSchema>;
export type CreateOrganizationInput = Infer<typeof createOrganizationSchema>;
export type UpdateOrganizationInput = Infer<typeof updateOrganizationSchema>;
export type OrganizationMember = Infer<typeof organizationMemberSchema>;
export type OrganizationInvitationRole = Infer<typeof organizationInvitationRoleSchema>;
export type OrganizationInvitationStatus = Infer<typeof organizationInvitationStatusSchema>;
export type OrganizationInvitation = Infer<typeof organizationInvitationSchema>;
export type InvitationEmailSendStatus = Infer<typeof invitationEmailSendStatusSchema>;
export type OrganizationInvitationCreateResult = Infer<
  typeof organizationInvitationCreateResultSchema
>;
export type OrganizationOwnershipTransfer = Infer<
  typeof organizationOwnershipTransferSchema
>;
export type CreateOrganizationInvitationInput = Infer<
  typeof createOrganizationInvitationSchema
>;
export type AcceptOrganizationInvitationInput = Infer<
  typeof acceptOrganizationInvitationSchema
>;
export type UpdateOrganizationMemberInput = Infer<
  typeof updateOrganizationMemberSchema
>;
export type TransferOrganizationOwnershipInput = Infer<
  typeof transferOrganizationOwnershipSchema
>;

export interface OrganizationPage {
  items: Organization[];
  meta: PaginationMeta;
  links: PaginationLinks;
}

export interface OrganizationMemberPage {
  items: OrganizationMember[];
  meta: PaginationMeta;
  links: PaginationLinks;
}

export interface OrganizationInvitationPage {
  items: OrganizationInvitation[];
  meta: PaginationMeta;
  links: PaginationLinks;
}
