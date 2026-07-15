import {
  array,
  int,
  iso,
  literal,
  maxLength,
  maximum,
  minLength,
  nonnegative,
  nullable,
  number,
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

const apiKeyScopePattern = /^(?:\*|[a-z][a-z0-9_-]{0,31}:[a-z][a-z0-9_-]{0,31})$/;
const safeIdSchema = number().check(int(), positive(), maximum(Number.MAX_SAFE_INTEGER));
const timestampSchema = iso.datetime({ offset: true });
const canonicalApiKeyNameSchema = string().check(
  refine(value => value === value.trim() && hasValidNameLength(value))
);
const apiKeyNameInputSchema = string().check(trim(), refine(hasValidNameLength));
const canonicalApiKeyScopeSchema = string().check(regex(apiKeyScopePattern));
const apiKeyScopeInputSchema = string().check(trim(), toLowerCase(), regex(apiKeyScopePattern));
const positiveIntegerSchema = number().check(int(), positive());
const nonnegativeIntegerSchema = number().check(int(), nonnegative());

export const apiKeySchema = strictObject({
  id: safeIdSchema,
  user_id: safeIdSchema,
  name: canonicalApiKeyNameSchema,
  key_prefix: string().check(regex(/^luas_[a-z0-9_-]+$/)),
  scopes: array(canonicalApiKeyScopeSchema).check(maxLength(32)),
  last_used_at: optional(timestampSchema),
  expires_at: optional(timestampSchema),
  revoked_at: optional(timestampSchema),
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

const paginationMetaSchema = strictObject({
  current_page: positiveIntegerSchema,
  per_page: positiveIntegerSchema,
  total: nonnegativeIntegerSchema,
  last_page: positiveIntegerSchema,
  from: nonnegativeIntegerSchema,
  to: nonnegativeIntegerSchema,
});

const paginationLinksSchema = strictObject({
  first: string(),
  last: string(),
  prev: nullable(string()),
  next: nullable(string()),
});

export const apiKeyPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(apiKeySchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const createApiKeyResultSchema = strictObject({
  api_key: apiKeySchema,
  plaintext_key: string().check(
    minLength(16),
    maxLength(512),
    refine(value => value === value.trim())
  ),
});

export const createApiKeySchema = strictObject({
  name: apiKeyNameInputSchema,
  scopes: array(apiKeyScopeInputSchema).check(maxLength(32)),
  expires_at: optional(timestampSchema),
});

export const apiKeyRouteIdSchema = pipe(
  pipe(string().check(regex(/^[1-9]\d*$/)), transform(Number)),
  safeIdSchema
);

function hasValidNameLength(value: string): boolean {
  const length = Array.from(value).length;
  return length >= 1 && length <= 100;
}
