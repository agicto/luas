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
  positive,
  record,
  refine,
  regex,
  strictObject,
  string,
  union,
} from 'zod/mini';

export const ASSET_MAX_BROWSER_BYTES = 100 * 1_024 * 1_024;
export const ASSET_DEFAULT_UPLOAD_BYTES = 10 * 1_024 * 1_024;

const assetIdPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const idempotencyPattern = /^[A-Za-z0-9._:-]{1,128}$/;
const mediaTypes = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'application/pdf',
  'text/plain',
  'text/csv',
] as const;

const assetIdSchema = string().check(regex(assetIdPattern));
const timestampSchema = iso.datetime({ offset: true });
const assetSizeSchema = number().check(int(), positive(), maximum(ASSET_MAX_BROWSER_BYTES));
const positiveIntegerSchema = number().check(int(), positive());
const nonnegativeIntegerSchema = number().check(int(), nonnegative());
const filenameSchema = string().check(
  minLength(1),
  maxLength(255),
  refine(value => isSafeFilename(value))
);
const mediaTypeSchema = union(mediaTypes.map(value => literal(value)));
const grantHeadersSchema = record(string(), string()).check(
  refine(headers => isSafeGrantHeaderRecord(headers))
);

export const assetSchema = strictObject({
  id: assetIdSchema,
  original_name: filenameSchema,
  media_type: mediaTypeSchema,
  size_bytes: assetSizeSchema,
  status: union([literal('pending'), literal('ready'), literal('rejected')]),
  created_at: timestampSchema,
  ready_at: nullable(timestampSchema),
});

export const transferGrantSchema = strictObject({
  method: union([literal('GET'), literal('PUT')]),
  url: string().check(
    minLength(1),
    maxLength(8_192),
    refine(value => !containsControl(value))
  ),
  headers: grantHeadersSchema,
  expires_at: timestampSchema,
});

export const uploadIntentResultSchema = strictObject({
  asset: assetSchema,
  upload: transferGrantSchema,
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

export const assetPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(assetSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const createUploadIntentSchema = strictObject({
  idempotency_key: string().check(regex(idempotencyPattern)),
  original_name: filenameSchema,
  media_type: mediaTypeSchema,
  size_bytes: assetSizeSchema,
});

export const deleteAssetResultSchema = strictObject({ deleted: literal(true) });
export const assetRouteIdSchema = assetIdSchema;
export const assetTransferTokenSchema = string().check(regex(/^[A-Za-z0-9._-]{32,256}$/));

export function isAllowedAssetFile(file: Pick<File, 'name' | 'size' | 'type'>): boolean {
  const parsed = createUploadIntentSchema.safeParse({
    idempotency_key: crypto.randomUUID(),
    original_name: file.name,
    media_type: file.type,
    size_bytes: file.size,
  });
  if (!parsed.success) return false;
  return extensionMatchesMediaType(file.name, file.type);
}

export function extensionMatchesMediaType(name: string, mediaType: string): boolean {
  const extension = name.includes('.') ? `.${name.split('.').pop()?.toLowerCase() ?? ''}` : '';
  const extensions: Record<string, readonly string[]> = {
    'image/jpeg': ['.jpg', '.jpeg'],
    'image/png': ['.png'],
    'image/webp': ['.webp'],
    'application/pdf': ['.pdf'],
    'text/plain': ['.txt'],
    'text/csv': ['.csv'],
  };
  return extensions[mediaType]?.includes(extension) ?? false;
}

function isSafeFilename(value: string): boolean {
  return (
    value === value.trim() &&
    value !== '.' &&
    value !== '..' &&
    !value.includes('/') &&
    !value.includes('\\') &&
    !containsControl(value)
  );
}

function isSafeGrantHeaderRecord(headers: Record<string, string>): boolean {
  const entries = Object.entries(headers);
  return (
    entries.length <= 16 &&
    entries.every(
      ([name, value]) =>
        /^[a-z0-9!#$%&'*+.^_`|~-]+$/.test(name) &&
        name === name.toLowerCase() &&
        value.length <= 2_048 &&
        !containsControl(value)
    )
  );
}

function containsControl(value: string): boolean {
  return /[\u0000-\u001F\u007F]/u.test(value);
}
