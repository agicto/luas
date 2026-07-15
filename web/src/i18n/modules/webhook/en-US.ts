import type { LocaleMessageShape } from '../../locale-message-shape';
import type { WebhookMessages } from './zh-Hans';

const messages = {
  title: 'Outbound webhooks',
  description:
    'Manage organization subscriptions, signing secrets, and minimized delivery records.',
  endpoints: {
    title: 'Endpoints',
    description: 'Each endpoint has an independent signing secret and a finite event subscription.',
    emptyTitle: 'No webhook endpoints',
    emptyDescription: 'Create an endpoint to send the fixed test event to a consumer integration.',
  },
  deliveries: {
    title: 'Delivery ledger',
    description: 'Records exclude target URLs, event payloads, signatures, and response bodies.',
    emptyTitle: 'No delivery records',
    emptyDescription: 'Minimized results appear here after an endpoint test is queued.',
  },
  columns: {
    endpoint: 'Endpoint',
    events: 'Events',
    status: 'Status',
    failures: 'Consecutive failures',
    updated: 'Updated',
    actions: 'Actions',
    message: 'Message',
    result: 'Result',
    attempts: 'Attempts',
    number: 'Number',
    duration: 'Duration',
    completed: 'Completed',
  },
  actions: {
    create: 'Create endpoint',
    edit: 'Edit endpoint',
    delete: 'Delete endpoint',
    rotate: 'Rotate signing secret',
    test: 'Send test event',
    attempts: 'View delivery attempts',
    retry: 'Retry',
    refresh: 'Refresh',
    previous: 'Previous',
    next: 'Next',
    cancel: 'Cancel',
    save: 'Save',
    copy: 'Copy',
    done: 'Done',
  },
  form: {
    createTitle: 'Create webhook endpoint',
    editTitle: 'Edit webhook endpoint',
    description:
      'The target must be a canonical HTTPS URL without credentials, query, or fragment.',
    name: 'Name',
    namePlaceholder: 'For example, Order processor',
    url: 'Target URL',
    urlPlaceholder: 'https://hooks.example.com/luas',
    eventTypes: 'Event types',
    invalid: 'Check the name, HTTPS target, and event selection.',
  },
  secret: {
    title: 'One-time signing secret',
    description: 'This plaintext secret cannot be viewed again after closing.',
    warningTitle: 'Store it securely now',
    warning:
      'Luas persists only encrypted material; lists and delivery records never return plaintext.',
    label: 'Signing secret',
    overlap: 'The previous secret will stop signing after {time}.',
  },
  delete: {
    title: 'Delete webhook endpoint',
    description: 'This removes “{name}” and cancels unfinished deliveries. It cannot be undone.',
  },
  rotate: {
    title: 'Rotate signing secret',
    description:
      'A new secret is generated for “{name}” with a bounded overlap for old signatures.',
  },
  attempts: {
    title: 'Delivery attempts',
    description: 'Minimized execution outcomes for message {messageId}.',
    empty:
      'This delivery has no network attempts. The development mock performs no outbound request.',
  },
  endpointStatus: {
    active: 'Active',
    disabled: 'Disabled',
  },
  deliveryStatus: {
    pending: 'Pending',
    processing: 'Processing',
    delivered: 'Delivered',
    failed: 'Failed',
    canceled: 'Canceled',
  },
  outcome: {
    delivered: 'Delivered',
    retryScheduled: 'Retry scheduled',
    failed: 'Failed',
  },
  messages: {
    created: 'Webhook endpoint created',
    updated: 'Webhook endpoint updated',
    deleted: 'Webhook endpoint deleted',
    enabled: 'Webhook endpoint enabled',
    disabled: 'Webhook endpoint disabled',
    rotated: 'Signing secret rotated',
    testQueued: 'Test delivery queued',
    copied: 'Signing secret copied',
    copyFailed: 'Could not copy the signing secret',
    page: 'Page {page} of {lastPage}',
  },
  errors: {
    generic: 'The webhook operation failed. Try again.',
    unavailable: 'The webhook service is temporarily unavailable.',
    forbidden: 'You cannot manage webhooks for this organization.',
    endpointNotFound: 'This webhook endpoint does not exist or was deleted.',
    deliveryNotFound: 'This delivery does not exist or belongs to another organization.',
    invalidTarget: 'The target URL violates the webhook network security policy.',
    invalidEventType: 'The selected event type is outside the finite catalog.',
    versionConflict: 'The endpoint changed elsewhere. The latest state has been loaded.',
    preconditionRequired: 'Refresh the endpoint before changing it.',
    replayNotAllowed: 'The current endpoint or delivery state does not allow this operation.',
    invalidResponse: 'The webhook service returned data this client cannot recognize.',
  },
} as const satisfies LocaleMessageShape<WebhookMessages>;

export default messages;
