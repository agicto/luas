import type { components } from '@/http/generated/openapi';

export type ApiKey = components['schemas']['APIKey'];

export interface ApiKeyPage {
  items: ApiKey[];
  meta: components['schemas']['PaginationMeta'];
  links: components['schemas']['PaginationLinks'];
}

export type CreateApiKeyInput = components['schemas']['CreateAPIKeyRequest'];
export type CreateApiKeyResult = components['schemas']['CreateAPIKeyResult'];
