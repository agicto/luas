import type { LocaleMessageShape } from '../../locale-message-shape';
import type { UsageMessages } from './zh-Hans';

const messages = {
  title: 'Usage',
  description: 'Review business usage and effective limits for the current UTC period.',
  user: {
    title: 'Personal usage',
    description: 'The finite metric catalog for the current account.',
  },
  organization: {
    title: 'Organization usage',
    description: 'The finite metric catalog for this organization, visible to managers only.',
  },
  columns: {
    metric: 'Metric',
    used: 'Used',
    limit: 'Limit',
    remaining: 'Remaining',
    period: 'UTC period',
  },
  metrics: {
    apiRequests: 'API requests',
    aiInputTokens: 'AI input tokens',
    aiOutputTokens: 'AI output tokens',
    assetTransferBytes: 'Asset transfer bytes',
    workflowRuns: 'Workflow runs',
  },
  units: {
    request: 'requests',
    token: 'tokens',
    byte: 'bytes',
    run: 'runs',
  },
  unlimited: 'Unlimited',
  notApplicable: 'N/A',
  overage: '{count} over',
  retry: 'Retry',
  errors: {
    forbidden: 'You do not have permission to view this usage.',
    generic: 'Usage could not be loaded.',
    invalidResponse: 'The usage service returned invalid data.',
    unavailable: 'The usage service is temporarily unavailable.',
  },
} as const satisfies LocaleMessageShape<UsageMessages>;

export default messages;
