import {
  array,
  boolean,
  int,
  iso,
  literal,
  maximum,
  maxLength,
  nonnegative,
  nullable,
  number,
  optional,
  positive,
  refine,
  regex,
  strictObject,
  string,
} from 'zod/mini';

const safeIdSchema = number().check(int(), positive(), maximum(Number.MAX_SAFE_INTEGER));
const nonnegativeIntegerSchema = number().check(
  int(),
  nonnegative(),
  maximum(Number.MAX_SAFE_INTEGER)
);
const positiveIntegerSchema = number().check(
  int(),
  positive(),
  maximum(Number.MAX_SAFE_INTEGER)
);
const timestampSchema = iso.datetime({ offset: true });
const kindSchema = string().check(
  maxLength(100),
  regex(/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$/)
);
const titleSchema = string().check(refine(value => hasBoundedText(value, 160)));
const bodySchema = string().check(refine(value => hasBoundedText(value, 4_000)));
const actionUrlSchema = string().check(maxLength(2_048), refine(isSafeActionUrl));

export const notificationSchema = strictObject({
  id: safeIdSchema,
  kind: kindSchema,
  title: titleSchema,
  body: bodySchema,
  action_url: optional(actionUrlSchema),
  is_read: boolean(),
  read_at: nullable(timestampSchema),
  created_at: timestampSchema,
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

export const notificationPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(notificationSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const notificationStatusSchema = strictObject({
  unread_count: nonnegativeIntegerSchema,
});

export const notificationReadStateResultSchema = strictObject({
  updated_count: nonnegativeIntegerSchema,
  unread_count: nonnegativeIntegerSchema,
});

export const notificationPreferenceSchema = strictObject({
  in_app_enabled: boolean(),
  email_enabled: boolean(),
});

export const replaceNotificationReadStateSchema = strictObject({
  is_read: boolean(),
});

export const markNotificationsReadSchema = strictObject({
  through_id: safeIdSchema,
});

export const notificationRouteIdSchema = string().check(regex(/^[1-9]\d*$/));

function hasBoundedText(value: string, maxLength: number): boolean {
  const length = Array.from(value).length;
  return value === value.trim() && length >= 1 && length <= maxLength;
}

function isSafeActionUrl(value: string): boolean {
  if (
    !value.startsWith('/') ||
    value.startsWith('//') ||
    value.includes('\\') ||
    containsControl(value)
  ) {
    return false;
  }
  try {
    const parsed = new URL(value, 'https://luas.invalid');
    const decodedPath = decodeURIComponent(parsed.pathname);
    return (
      parsed.origin === 'https://luas.invalid' &&
      !decodedPath.startsWith('//') &&
      !decodedPath.includes('\\') &&
      !containsControl(decodedPath)
    );
  } catch {
    return false;
  }
}

function containsControl(value: string): boolean {
  return /[\u0000-\u001F\u007F]/u.test(value);
}
