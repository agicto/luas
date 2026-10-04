import type { z } from 'zod/mini';
import {
  assertContract,
  type Accepts,
  type ContractData,
  type ContractRequest,
  type ContractResponse,
  type Sends,
} from '@/http/contract';
import { ClientErrorCode } from '@/http/codes';
import request, { ApiError } from '@/http/request';
import { apiKeyPageEnvelopeSchema, createApiKeyResultSchema } from '@/features/api-key/schemas';
import type { ApiKeyPage, CreateApiKeyInput, CreateApiKeyResult } from '@/features/api-key/types';

export const apiKeyService = {
  async list(): Promise<ApiKeyPage> {
    const value = await request.getEnvelope<unknown>('/api-keys', {
      params: { page: 1, per_page: 100 },
    });
    return parseApiKeyPageResponse(value);
  },

  async create(input: CreateApiKeyInput): Promise<CreateApiKeyResult> {
    const value = await request.post<unknown, CreateApiKeyInput>('/api-keys', input);
    return parseCreateApiKeyResponse(value);
  },

  async revoke(apiKeyId: number): Promise<void> {
    await request.delete<void>(`/api-keys/${apiKeyId}`);
  },
};

export function parseApiKeyPageResponse(value: unknown): ApiKeyPage {
  const parsed = apiKeyPageEnvelopeSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return {
    items: parsed.data.data,
    meta: parsed.data.meta,
    links: parsed.data.links,
  };
}

export function parseCreateApiKeyResponse(value: unknown): CreateApiKeyResult {
  const parsed = createApiKeyResultSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function invalidResponse(): ApiError {
  return new ApiError(
    'API key service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}

assertContract<
  Accepts<z.input<typeof apiKeyPageEnvelopeSchema>, ContractResponse<'listApiKeys', 200>>
>();
assertContract<
  Accepts<z.input<typeof createApiKeyResultSchema>, ContractData<'createApiKey', 201>>
>();
assertContract<Sends<CreateApiKeyInput, ContractRequest<'createApiKey'>>>();
