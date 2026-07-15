import {
  array,
  enum as enumSchema,
  int,
  iso,
  literal,
  maximum,
  nonnegative,
  nullable,
  number,
  object,
  optional,
  pipe,
  positive,
  refine,
  regex,
  string,
  transform,
  trim,
} from 'zod/mini';

export const organizationRoleSchema = enumSchema(['owner', 'admin', 'member']);

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

export const createOrganizationSchema = object({
  name: organizationNameInputSchema,
  slug: optional(organizationSlugSchema),
});

export const updateOrganizationSchema = object({
  name: organizationNameInputSchema,
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
