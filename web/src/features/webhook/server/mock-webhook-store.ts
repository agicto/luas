import 'server-only';

import { randomBytes, randomUUID } from 'node:crypto';

import type {
  WebhookAttempt,
  WebhookDelivery,
  WebhookEndpoint,
  WebhookEndpointInput,
  WebhookEndpointSecret,
  WebhookPage,
} from '@/features/webhook/types';

export type MockWebhookStoreError =
  'delivery_not_found' | 'endpoint_not_found' | 'replay_not_allowed' | 'version_conflict';

interface MockWebhookDependencies {
  now: () => Date;
  randomBytes: (size: number) => Buffer;
  randomUUID: () => string;
}

interface ListOptions {
  page: number;
  perPage: number;
}

interface DeliveryListOptions extends ListOptions {
  endpointId?: number;
  status?: WebhookDelivery['status'];
}

export class MockWebhookStore {
  private endpoints: WebhookEndpoint[] = [];
  private deliveries: WebhookDelivery[] = [];
  private deliveryOrganizations = new Map<number, number>();
  private attemptsByDelivery = new Map<number, WebhookAttempt[]>();
  private testDeliveries = new Map<string, number>();
  private nextEndpointId = 1;
  private nextDeliveryId = 1;

  constructor(
    private readonly dependencies: MockWebhookDependencies = {
      now: () => new Date(),
      randomBytes,
      randomUUID,
    }
  ) {}

  reset(): void {
    this.endpoints = [];
    this.deliveries = [];
    this.deliveryOrganizations.clear();
    this.attemptsByDelivery.clear();
    this.testDeliveries.clear();
    this.nextEndpointId = 1;
    this.nextDeliveryId = 1;
  }

  eventTypes(): readonly ['webhook.test'] {
    return ['webhook.test'];
  }

  listEndpoints(
    organizationId: number,
    options: ListOptions,
    path = '/api/webhook-endpoints'
  ): WebhookPage<WebhookEndpoint> {
    const values = this.endpoints
      .filter(endpoint => endpoint.organization_id === organizationId)
      .sort(newestFirst)
      .map(cloneEndpoint);
    return paginate(values, options, path);
  }

  createEndpoint(organizationId: number, input: WebhookEndpointInput): WebhookEndpointSecret {
    const now = this.dependencies.now().toISOString();
    const signingSecret = this.generateSecret();
    const endpoint: WebhookEndpoint = {
      id: this.nextEndpointId++,
      organization_id: organizationId,
      name: input.name,
      url: input.url,
      event_types: [...input.event_types],
      status: 'active',
      disabled_reason: '',
      consecutive_failures: 0,
      version: 1,
      secret_hint: signingSecret.slice(-8),
      secret_version: 1,
      previous_secret_expiry: null,
      created_at: now,
      updated_at: now,
    };
    this.endpoints.push(endpoint);
    return {
      endpoint: cloneEndpoint(endpoint),
      signing_secret: signingSecret,
      previous_secret_expiry: null,
    };
  }

  updateEndpoint(
    organizationId: number,
    endpointId: number,
    expectedVersion: number,
    input: WebhookEndpointInput
  ): WebhookEndpoint | MockWebhookStoreError {
    const endpoint = this.endpoint(organizationId, endpointId);
    if (!endpoint) return 'endpoint_not_found';
    if (endpoint.version !== expectedVersion) return 'version_conflict';
    endpoint.name = input.name;
    endpoint.url = input.url;
    endpoint.event_types = [...input.event_types];
    this.touch(endpoint);
    return cloneEndpoint(endpoint);
  }

  replaceEndpointStatus(
    organizationId: number,
    endpointId: number,
    expectedVersion: number,
    enabled: boolean
  ): WebhookEndpoint | MockWebhookStoreError {
    const endpoint = this.endpoint(organizationId, endpointId);
    if (!endpoint) return 'endpoint_not_found';
    if (endpoint.version !== expectedVersion) return 'version_conflict';
    endpoint.status = enabled ? 'active' : 'disabled';
    endpoint.disabled_reason = enabled ? '' : 'manual';
    endpoint.consecutive_failures = 0;
    this.touch(endpoint);
    return cloneEndpoint(endpoint);
  }

  deleteEndpoint(
    organizationId: number,
    endpointId: number,
    expectedVersion: number
  ): true | MockWebhookStoreError {
    const endpoint = this.endpoint(organizationId, endpointId);
    if (!endpoint) return 'endpoint_not_found';
    if (endpoint.version !== expectedVersion) return 'version_conflict';
    this.endpoints = this.endpoints.filter(value => value !== endpoint);
    return true;
  }

  rotateEndpointSecret(
    organizationId: number,
    endpointId: number,
    expectedVersion: number
  ): WebhookEndpointSecret | MockWebhookStoreError {
    const endpoint = this.endpoint(organizationId, endpointId);
    if (!endpoint) return 'endpoint_not_found';
    if (endpoint.version !== expectedVersion) return 'version_conflict';
    const signingSecret = this.generateSecret();
    const previousSecretExpiry = new Date(
      this.dependencies.now().getTime() + 24 * 60 * 60 * 1_000
    ).toISOString();
    endpoint.secret_hint = signingSecret.slice(-8);
    endpoint.secret_version += 1;
    endpoint.previous_secret_expiry = previousSecretExpiry;
    this.touch(endpoint);
    return {
      endpoint: cloneEndpoint(endpoint),
      signing_secret: signingSecret,
      previous_secret_expiry: previousSecretExpiry,
    };
  }

  testEndpoint(
    organizationId: number,
    endpointId: number,
    idempotencyKey: string
  ): WebhookDelivery | MockWebhookStoreError {
    const endpoint = this.endpoint(organizationId, endpointId);
    if (!endpoint) return 'endpoint_not_found';
    if (endpoint.status !== 'active') return 'replay_not_allowed';

    const key = `${organizationId}:${endpointId}:${idempotencyKey}`;
    const existingId = this.testDeliveries.get(key);
    if (existingId !== undefined) {
      const existing = this.deliveries.find(delivery => delivery.id === existingId);
      return existing ? cloneDelivery(existing) : 'delivery_not_found';
    }

    const now = this.dependencies.now().toISOString();
    const delivery: WebhookDelivery = {
      id: this.nextDeliveryId++,
      endpoint_id: endpointId,
      message_id: `msg_${this.dependencies.randomUUID().replaceAll('-', '')}`,
      event_type: 'webhook.test',
      status: 'canceled',
      attempt_count: 0,
      replay_count: 0,
      http_status: null,
      failure_code: 'WEBHOOK.MOCK_NOT_DELIVERED',
      response_truncated: false,
      available_at: now,
      delivered_at: null,
      created_at: now,
      updated_at: now,
    };
    this.deliveries.push(delivery);
    this.deliveryOrganizations.set(delivery.id, organizationId);
    this.attemptsByDelivery.set(delivery.id, []);
    this.testDeliveries.set(key, delivery.id);
    return cloneDelivery(delivery);
  }

  listDeliveries(
    organizationId: number,
    options: DeliveryListOptions,
    path = '/api/webhook-deliveries'
  ): WebhookPage<WebhookDelivery> {
    const values = this.deliveries
      .filter(delivery => this.deliveryOrganizations.get(delivery.id) === organizationId)
      .filter(
        delivery => options.endpointId === undefined || delivery.endpoint_id === options.endpointId
      )
      .filter(delivery => options.status === undefined || delivery.status === options.status)
      .sort(newestFirst)
      .map(cloneDelivery);
    return paginate(values, options, path);
  }

  listAttempts(
    organizationId: number,
    deliveryId: number,
    options: ListOptions,
    path = `/api/webhook-deliveries/${deliveryId}/attempts`
  ): WebhookPage<WebhookAttempt> | MockWebhookStoreError {
    const delivery = this.deliveries.find(value => value.id === deliveryId);
    if (!delivery || this.deliveryOrganizations.get(delivery.id) !== organizationId) {
      return 'delivery_not_found';
    }
    const values = [...(this.attemptsByDelivery.get(deliveryId) ?? [])]
      .sort((left, right) => right.number - left.number)
      .map(cloneAttempt);
    return paginate(values, options, path);
  }

  private endpoint(organizationId: number, endpointId: number): WebhookEndpoint | undefined {
    return this.endpoints.find(
      endpoint => endpoint.id === endpointId && endpoint.organization_id === organizationId
    );
  }

  private touch(endpoint: WebhookEndpoint): void {
    endpoint.version += 1;
    endpoint.updated_at = this.dependencies.now().toISOString();
  }

  private generateSecret(): string {
    return `whsec_${this.dependencies.randomBytes(32).toString('base64')}`;
  }
}

export const mockWebhookStore = new MockWebhookStore();

function paginate<T>(
  values: readonly T[],
  { page, perPage }: ListOptions,
  path: string
): WebhookPage<T> {
  const total = values.length;
  const lastPage = Math.max(Math.ceil(total / perPage), 1);
  const offset = (page - 1) * perPage;
  const items = values.slice(offset, offset + perPage);
  const query = (targetPage: number) => `${path}?page=${targetPage}&per_page=${perPage}`;
  return {
    items,
    meta: {
      current_page: page,
      per_page: perPage,
      total,
      last_page: lastPage,
      from: items.length === 0 ? 0 : offset + 1,
      to: items.length === 0 ? 0 : offset + items.length,
    },
    links: {
      first: query(1),
      last: query(lastPage),
      prev: page > 1 ? query(page - 1) : null,
      next: page < lastPage ? query(page + 1) : null,
    },
  };
}

function newestFirst(
  left: { created_at: string; id: number },
  right: { created_at: string; id: number }
) {
  return right.created_at.localeCompare(left.created_at) || right.id - left.id;
}

function cloneEndpoint(value: WebhookEndpoint): WebhookEndpoint {
  return { ...value, event_types: [...value.event_types] };
}

function cloneDelivery(value: WebhookDelivery): WebhookDelivery {
  return { ...value };
}

function cloneAttempt(value: WebhookAttempt): WebhookAttempt {
  return { ...value };
}
