import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import {
  webhookAttemptPageEnvelopeSchema,
  webhookDeliveryPageEnvelopeSchema,
  webhookDeliverySchema,
  webhookEndpointPageEnvelopeSchema,
  webhookEndpointSchema,
  webhookEndpointSecretSchema,
  webhookEventTypeListSchema,
  webhookIdempotencyKeySchema,
} from '@/features/webhook/schemas';
import type {
  WebhookAttemptPage,
  WebhookDelivery,
  WebhookDeliveryPage,
  WebhookEndpoint,
  WebhookEndpointInput,
  WebhookEndpointPage,
  WebhookEndpointSecret,
} from '@/features/webhook/types';

interface PageOptions {
  page?: number;
  perPage?: number;
}

interface DeliveryOptions extends PageOptions {
  endpointId?: number;
}

export const webhookService = {
  async eventTypes(organizationId: number): Promise<string[]> {
    validOrganizationId(organizationId);
    const value = await request.get<unknown>('/webhook-event-types', {
      headers: organizationHeaders(organizationId),
    });
    const parsed = webhookEventTypeListSchema.safeParse(value);
    if (!parsed.success || parsed.data.length !== 1 || parsed.data[0] !== 'webhook.test') {
      throw invalidResponse();
    }
    return parsed.data;
  },

  async endpoints(
    organizationId: number,
    { page = 1, perPage = 100 }: PageOptions = {}
  ): Promise<WebhookEndpointPage> {
    validOrganizationId(organizationId);
    const value = await request.getEnvelope<unknown>('/webhook-endpoints', {
      headers: organizationHeaders(organizationId),
      params: { page, per_page: perPage },
    });
    return parsePage(value, webhookEndpointPageEnvelopeSchema) as WebhookEndpointPage;
  },

  async create(
    organizationId: number,
    input: WebhookEndpointInput
  ): Promise<WebhookEndpointSecret> {
    const value = await request.post<unknown, WebhookEndpointInput>('/webhook-endpoints', input, {
      headers: organizationHeaders(organizationId),
    });
    return parseSecret(value);
  },

  async update(
    organizationId: number,
    endpointId: number,
    input: WebhookEndpointInput,
    expectedVersion: number
  ): Promise<WebhookEndpoint> {
    const value = await request.patch<unknown, WebhookEndpointInput>(
      `/webhook-endpoints/${endpointId}`,
      input,
      { headers: mutationHeaders(organizationId, expectedVersion) }
    );
    return parseEndpoint(value);
  },

  async replaceStatus(
    organizationId: number,
    endpointId: number,
    enabled: boolean,
    expectedVersion: number
  ): Promise<WebhookEndpoint> {
    const value = await request.put<unknown, { enabled: boolean }>(
      `/webhook-endpoints/${endpointId}/status`,
      { enabled },
      { headers: mutationHeaders(organizationId, expectedVersion) }
    );
    return parseEndpoint(value);
  },

  async remove(organizationId: number, endpointId: number, expectedVersion: number): Promise<void> {
    await request.delete<void>(`/webhook-endpoints/${endpointId}`, {
      headers: mutationHeaders(organizationId, expectedVersion),
    });
  },

  async rotate(
    organizationId: number,
    endpointId: number,
    expectedVersion: number
  ): Promise<WebhookEndpointSecret> {
    const value = await request.post<unknown>(
      `/webhook-endpoints/${endpointId}/secret-rotations`,
      undefined,
      { headers: mutationHeaders(organizationId, expectedVersion) }
    );
    return parseSecret(value);
  },

  async test(
    organizationId: number,
    endpointId: number,
    idempotencyKey: string
  ): Promise<WebhookDelivery> {
    if (!webhookIdempotencyKeySchema.safeParse(idempotencyKey).success) throw invalidResponse();
    const value = await request.post<unknown>(`/webhook-endpoints/${endpointId}/tests`, undefined, {
      headers: {
        ...organizationHeaders(organizationId),
        'Idempotency-Key': idempotencyKey,
      },
    });
    const parsed = webhookDeliverySchema.safeParse(value);
    if (!parsed.success || parsed.data.endpoint_id !== endpointId) throw invalidResponse();
    return parsed.data;
  },

  async deliveries(
    organizationId: number,
    { page = 1, perPage = 100, endpointId }: DeliveryOptions = {}
  ): Promise<WebhookDeliveryPage> {
    validOrganizationId(organizationId);
    const value = await request.getEnvelope<unknown>('/webhook-deliveries', {
      headers: organizationHeaders(organizationId),
      params: { page, per_page: perPage, ...(endpointId ? { endpoint_id: endpointId } : {}) },
    });
    return parsePage(value, webhookDeliveryPageEnvelopeSchema) as WebhookDeliveryPage;
  },

  async attempts(
    organizationId: number,
    deliveryId: number,
    { page = 1, perPage = 100 }: PageOptions = {}
  ): Promise<WebhookAttemptPage> {
    validOrganizationId(organizationId);
    const value = await request.getEnvelope<unknown>(`/webhook-deliveries/${deliveryId}/attempts`, {
      headers: organizationHeaders(organizationId),
      params: { page, per_page: perPage },
    });
    return parsePage(value, webhookAttemptPageEnvelopeSchema) as WebhookAttemptPage;
  },
};

function parseEndpoint(value: unknown): WebhookEndpoint {
  const parsed = webhookEndpointSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function parseSecret(value: unknown): WebhookEndpointSecret {
  const parsed = webhookEndpointSecretSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function parsePage(
  value: unknown,
  schema:
    | typeof webhookEndpointPageEnvelopeSchema
    | typeof webhookDeliveryPageEnvelopeSchema
    | typeof webhookAttemptPageEnvelopeSchema
) {
  const parsed = schema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return { items: parsed.data.data, meta: parsed.data.meta, links: parsed.data.links };
}

function organizationHeaders(organizationId: number): Record<string, string> {
  validOrganizationId(organizationId);
  return { 'Organization-Id': String(organizationId) };
}

function mutationHeaders(organizationId: number, version: number): Record<string, string> {
  if (!Number.isSafeInteger(version) || version < 1) throw invalidResponse();
  return {
    ...organizationHeaders(organizationId),
    'If-Match': `"webhook-endpoint-v${version}"`,
  };
}

function validOrganizationId(value: number): void {
  if (!Number.isSafeInteger(value) || value < 1) throw invalidResponse();
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Webhook service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}
