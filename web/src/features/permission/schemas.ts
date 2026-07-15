import {
  array,
  boolean,
  int,
  iso,
  literal,
  maxLength,
  maximum,
  number,
  positive,
  refine,
  regex,
  strictObject,
  string,
  trim,
} from 'zod/mini';

import {
  paginationLinksSchema,
  paginationMetaSchema,
} from '@/features/organization/schemas';

const safeIdSchema = number().check(
  int(),
  positive(),
  maximum(Number.MAX_SAFE_INTEGER)
);
const timestampSchema = iso.datetime({ offset: true });
const permissionKeySchema = string().check(
  maxLength(100),
  regex(/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$/)
);
const canonicalRoleNameSchema = string().check(
  refine((value) => value === value.trim() && validRoleName(value))
);
const roleNameInputSchema = string().check(trim(), refine(validRoleName));
const accessRoleSlugSchema = string().check(
  regex(/^[a-z0-9](?:[a-z0-9-]{1,48}[a-z0-9])$/)
);
const permissionListSchema = array(permissionKeySchema).check(
  maxLength(100),
  refine(uniqueValues)
);
const roleIdListSchema = array(safeIdSchema).check(
  maxLength(100),
  refine(uniqueValues)
);

export const permissionContextSchema = strictObject({
  organization_id: safeIdSchema,
  membership_id: safeIdSchema,
  is_owner: boolean(),
  access_role_ids: roleIdListSchema,
  permissions: permissionListSchema,
});

export const accessRoleSchema = strictObject({
  id: safeIdSchema,
  organization_id: safeIdSchema,
  name: canonicalRoleNameSchema,
  slug: accessRoleSlugSchema,
  permissions: permissionListSchema,
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const permissionCatalogSchema = strictObject({
  permissions: permissionListSchema,
});

export const memberAccessRolesSchema = strictObject({
  member_id: safeIdSchema,
  access_role_ids: roleIdListSchema,
});

export const createAccessRoleSchema = strictObject({
  name: roleNameInputSchema,
  slug: accessRoleSlugSchema,
  permissions: permissionListSchema,
});

export const updateAccessRoleSchema = strictObject({
  name: roleNameInputSchema,
  permissions: permissionListSchema,
});

export const replaceMemberAccessRolesSchema = strictObject({
  access_role_ids: roleIdListSchema,
});

export const accessRolePageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(accessRoleSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

function validRoleName(value: string): boolean {
  const length = Array.from(value).length;
  return length >= 2 && length <= 100;
}

function uniqueValues(values: readonly unknown[]): boolean {
  return new Set(values).size === values.length;
}
