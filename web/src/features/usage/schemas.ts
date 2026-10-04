import {
  array,
  boolean,
  int,
  iso,
  literal,
  maxLength,
  maximum,
  nonnegative,
  nullable,
  number,
  object,
  refine,
  strictObject,
  string,
  union,
} from 'zod/mini';

export const usageMetricSchema = union([
  literal('api.requests'),
  literal('ai.input_tokens'),
  literal('ai.output_tokens'),
  literal('asset.transfer_bytes'),
  literal('workflow.runs'),
]);

const safeIntegerSchema = number().check(int(), nonnegative(), maximum(Number.MAX_SAFE_INTEGER));
const nullableSafeIntegerSchema = nullable(safeIntegerSchema);

export const usageSummarySchema = strictObject({
  scope: union([literal('user'), literal('organization')]),
  metric: usageMetricSchema,
  unit: union([literal('request'), literal('token'), literal('byte'), literal('run')]),
  period: literal('month'),
  period_start: iso.datetime({ offset: true }),
  period_end: iso.datetime({ offset: true }),
  used: safeIntegerSchema,
  limit: nullableSafeIntegerSchema,
  remaining: nullableSafeIntegerSchema,
  overage: safeIntegerSchema,
  over_limit: boolean(),
  quota_source: union([literal('default'), literal('override')]),
  quota_version: safeIntegerSchema,
  updated_at: nullable(iso.datetime({ offset: true })),
}).check(refine(isSemanticallyValidSummary));

/**
 * Any well-formed summary as `contracts/openapi.yaml` describes it. The metric catalog is
 * code-owned and downstream apps extend it, so summaries are read with this schema first and only
 * the metrics this client renders are narrowed with `usageSummarySchema`.
 */
export const usageSummaryWireSchema = object({
  scope: union([literal('user'), literal('organization')]),
  metric: string(),
  unit: string(),
  period: union([literal('day'), literal('month')]),
  period_start: string(),
  period_end: string(),
  used: safeIntegerSchema,
  limit: nullableSafeIntegerSchema,
  remaining: nullableSafeIntegerSchema,
  overage: safeIntegerSchema,
  over_limit: boolean(),
  quota_source: union([literal('default'), literal('override')]),
  quota_version: safeIntegerSchema,
  updated_at: nullable(string()),
});

export const usageSummaryWireListSchema = array(usageSummaryWireSchema).check(
  maxLength(64),
  refine(values => {
    const identities = values.map(value => `${value.scope}:${value.metric}`);
    return new Set(identities).size === identities.length;
  })
);

function isSemanticallyValidSummary(summary: {
  metric:
    | 'api.requests'
    | 'ai.input_tokens'
    | 'ai.output_tokens'
    | 'asset.transfer_bytes'
    | 'workflow.runs';
  unit: 'request' | 'token' | 'byte' | 'run';
  period_start: string;
  period_end: string;
  used: number;
  limit: number | null;
  remaining: number | null;
  overage: number;
  over_limit: boolean;
  quota_source: 'default' | 'override';
  quota_version: number;
}): boolean {
  const expectedUnit = {
    'api.requests': 'request',
    'ai.input_tokens': 'token',
    'ai.output_tokens': 'token',
    'asset.transfer_bytes': 'byte',
    'workflow.runs': 'run',
  } as const;
  if (summary.unit !== expectedUnit[summary.metric]) return false;
  if (Date.parse(summary.period_end) <= Date.parse(summary.period_start)) return false;
  if (
    summary.quota_source === 'override' &&
    (summary.limit === null || summary.quota_version < 1)
  ) {
    return false;
  }
  if (summary.limit === null) {
    return summary.remaining === null && summary.overage === 0 && !summary.over_limit;
  }
  const overage = Math.max(summary.used - summary.limit, 0);
  const remaining = Math.max(summary.limit - summary.used, 0);
  return (
    summary.remaining === remaining &&
    summary.overage === overage &&
    summary.over_limit === overage > 0
  );
}
