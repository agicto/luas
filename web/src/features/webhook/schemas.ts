import {
  array,
  boolean,
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
  refine,
  regex,
  strictObject,
  string,
  union,
} from 'zod/mini';

const safeIdSchema = number().check(int(), positive(), maximum(Number.MAX_SAFE_INTEGER));
const nonnegativeIntegerSchema = number().check(
  int(),
  nonnegative(),
  maximum(Number.MAX_SAFE_INTEGER)
);
const timestampSchema = iso.datetime({ offset: true });
const webhookEventTypeSchema = literal('webhook.test');
const endpointNameSchema = string().check(
  minLength(1),
  maxLength(100),
  refine(value => value === value.trim() && !containsControl(value))
);
const targetUrlSchema = string().check(minLength(1), maxLength(2_048), refine(isSafeWebhookTarget));

export const webhookEndpointSchema = strictObject({
  id: safeIdSchema,
  organization_id: safeIdSchema,
  name: endpointNameSchema,
  url: targetUrlSchema,
  event_types: array(webhookEventTypeSchema).check(
    minLength(1),
    maxLength(128),
    refine(values => new Set(values).size === values.length)
  ),
  status: union([literal('active'), literal('disabled')]),
  disabled_reason: string().check(maxLength(64)),
  consecutive_failures: nonnegativeIntegerSchema,
  version: safeIdSchema,
  secret_hint: string().check(minLength(1), maxLength(16)),
  secret_version: safeIdSchema,
  previous_secret_expiry: nullable(timestampSchema),
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const webhookEndpointSecretSchema = strictObject({
  endpoint: webhookEndpointSchema,
  signing_secret: string().check(regex(/^whsec_[A-Za-z0-9+/]+={0,2}$/), maxLength(128)),
  previous_secret_expiry: nullable(timestampSchema),
});

export const webhookDeliverySchema = strictObject({
  id: safeIdSchema,
  endpoint_id: safeIdSchema,
  message_id: string().check(regex(/^msg_[A-Za-z0-9_-]{1,60}$/)),
  event_type: webhookEventTypeSchema,
  status: union([
    literal('pending'),
    literal('processing'),
    literal('delivered'),
    literal('failed'),
    literal('canceled'),
  ]),
  attempt_count: nonnegativeIntegerSchema,
  replay_count: nonnegativeIntegerSchema,
  http_status: nullable(number().check(int(), positive(), maximum(599))),
  failure_code: string().check(maxLength(64)),
  response_truncated: boolean(),
  available_at: timestampSchema,
  delivered_at: nullable(timestampSchema),
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const webhookAttemptSchema = strictObject({
  id: safeIdSchema,
  delivery_id: safeIdSchema,
  number: safeIdSchema,
  outcome: union([literal('delivered'), literal('retry_scheduled'), literal('failed')]),
  http_status: nullable(number().check(int(), positive(), maximum(599))),
  failure_code: string().check(maxLength(64)),
  duration_ms: nonnegativeIntegerSchema,
  response_truncated: boolean(),
  started_at: timestampSchema,
  completed_at: timestampSchema,
});

const paginationMetaSchema = strictObject({
  current_page: safeIdSchema,
  per_page: safeIdSchema,
  total: nonnegativeIntegerSchema,
  last_page: safeIdSchema,
  from: nonnegativeIntegerSchema,
  to: nonnegativeIntegerSchema,
});

const paginationLinksSchema = strictObject({
  first: string(),
  last: string(),
  prev: nullable(string()),
  next: nullable(string()),
});

export const webhookEndpointPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(webhookEndpointSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const webhookDeliveryPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(webhookDeliverySchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const webhookAttemptPageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(webhookAttemptSchema),
  meta: paginationMetaSchema,
  links: paginationLinksSchema,
});

export const webhookEventTypeListSchema = array(webhookEventTypeSchema).check(
  minLength(1),
  maxLength(128),
  refine(values => new Set(values).size === values.length)
);

export const webhookEndpointInputSchema = strictObject({
  name: endpointNameSchema,
  url: targetUrlSchema,
  event_types: array(webhookEventTypeSchema).check(
    minLength(1),
    maxLength(128),
    refine(values => new Set(values).size === values.length)
  ),
});

export const webhookEndpointStatusInputSchema = strictObject({ enabled: boolean() });
export const webhookIdempotencyKeySchema = string().check(
  regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/)
);
export const webhookRouteIdSchema = string().check(regex(/^[1-9]\d{0,15}$/));

function isSafeWebhookTarget(value: string): boolean {
  if (value !== value.trim() || containsControl(value) || value.includes('\\')) return false;
  try {
    const parsed = new URL(value);
    return (
      parsed.protocol === 'https:' &&
      parsed.username === '' &&
      parsed.password === '' &&
      parsed.search === '' &&
      parsed.hash === '' &&
      parsed.hostname !== '' &&
      !parsed.hostname.endsWith('.')
    );
  } catch {
    return false;
  }
}

function containsControl(value: string): boolean {
  return /[\u0000-\u001F\u007F]/u.test(value);
}
