import {
  array,
  email,
  enum as enumSchema,
  int,
  iso,
  literal,
  maxLength,
  maximum,
  minLength,
  nonnegative,
  nullable,
  number,
  object,
  optional,
  pipe,
  positive,
  refine,
  regex,
  strictObject,
  string,
  toLowerCase,
  transform,
  trim,
} from 'zod/mini';

export const organizationRoleSchema = enumSchema(['owner', 'admin', 'member']);
export const organizationInvitationRoleSchema = enumSchema(['admin', 'member']);
export const organizationInvitationStatusSchema = enumSchema([
  'pending',
  'accepted',
  'revoked',
  'expired',
]);
export const invitationEmailSendStatusSchema = enumSchema([
  'accepted_by_provider',
  'failed',
  'not_configured',
]);

const safeIdSchema = number().check(
  int(),
  positive(),
  maximum(Number.MAX_SAFE_INTEGER)
);
const timestampSchema = iso.datetime({ offset: true });
const positiveIntegerSchema = number().check(int(), positive());
const nonnegativeIntegerSchema = number().check(int(), nonnegative());
const organizationNameInputSchema = string().check(
  trim(),
  refine(hasValidOrganizationNameLength)
);
const organizationNameSchema = string().check(
  refine(
    (value) => value === value.trim() && hasValidOrganizationNameLength(value)
  )
);
const organizationSlugSchema = string().check(
  regex(/^[a-z0-9](?:[a-z0-9-]{1,48}[a-z0-9])$/)
);
const canonicalInvitationEmailSchema = email().check(
  maxLength(100),
  refine((value) => value === value.trim() && value === value.toLowerCase())
);
const invitationEmailInputSchema = pipe(
  string().check(trim(), toLowerCase(), maxLength(100)),
  email()
);
const invitationTokenSchema = string().check(
  trim(),
  minLength(1),
  maxLength(256)
);

export const organizationSchema = object({
  id: safeIdSchema,
  name: organizationNameSchema,
  slug: organizationSlugSchema,
  role: organizationRoleSchema,
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const organizationContextSchema = object({
  organization_id: safeIdSchema,
  organization_name: organizationNameSchema,
  organization_slug: organizationSlugSchema,
  membership_id: safeIdSchema,
  user_id: safeIdSchema,
  role: organizationRoleSchema,
});

export const paginationMetaSchema = object({
  current_page: positiveIntegerSchema,
  per_page: positiveIntegerSchema,
  total: nonnegativeIntegerSchema,
  last_page: positiveIntegerSchema,
  from: nonnegativeIntegerSchema,
  to: nonnegativeIntegerSchema,
});

export const paginationLinksSchema = object({
  first: string(),
  last: string(),
  prev: nullable(string()),
  next: nullable(string()),
});

export const organizationPageEnvelopeSchema = object({
  code: literal(0),
  message: string(),
  data: array(organizationSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const organizationMemberSchema = strictObject({
  id: safeIdSchema,
  user_id: safeIdSchema,
  username: string().check(minLength(1), maxLength(50)),
  nickname: optional(string().check(maxLength(50))),
  avatar: optional(string().check(maxLength(255))),
  role: organizationRoleSchema,
  joined_at: timestampSchema,
  updated_at: timestampSchema,
});

export const organizationMemberPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(organizationMemberSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const organizationInvitationSchema = strictObject({
  id: safeIdSchema,
  organization_id: safeIdSchema,
  email: canonicalInvitationEmailSchema,
  role: organizationInvitationRoleSchema,
  status: organizationInvitationStatusSchema,
  expires_at: timestampSchema,
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const organizationInvitationPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(organizationInvitationSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const organizationInvitationCreateResultSchema = strictObject({
  invitation: organizationInvitationSchema,
  email_send_status: invitationEmailSendStatusSchema,
});

export const organizationOwnershipTransferSchema = strictObject({
  previous_owner: organizationMemberSchema,
  new_owner: organizationMemberSchema,
});

export const createOrganizationSchema = object({
  name: organizationNameInputSchema,
  slug: optional(organizationSlugSchema),
});

export const updateOrganizationSchema = object({
  name: organizationNameInputSchema,
});

export const createOrganizationInvitationSchema = strictObject({
  email: invitationEmailInputSchema,
  role: organizationInvitationRoleSchema,
});

export const acceptOrganizationInvitationSchema = strictObject({
  token: invitationTokenSchema,
});

export const updateOrganizationMemberSchema = strictObject({
  role: organizationInvitationRoleSchema,
});

export const transferOrganizationOwnershipSchema = strictObject({
  new_owner_member_id: safeIdSchema,
});

export const organizationRouteIdSchema = pipe(
  pipe(
    string().check(regex(/^[1-9]\d*$/)),
    transform(Number)
  ),
  safeIdSchema
);

function hasValidOrganizationNameLength(value: string): boolean {
  const length = Array.from(value).length;
  return length >= 2 && length <= 100;
}
