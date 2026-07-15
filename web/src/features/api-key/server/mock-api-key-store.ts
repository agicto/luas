import 'server-only';

import { createHash, randomBytes } from 'node:crypto';

import type { AuthUser } from '@/features/auth/types';
import type {
  ApiKey,
  ApiKeyPage,
  CreateApiKeyInput,
  CreateApiKeyResult,
} from '@/features/api-key/types';

interface MockApiKeyRecord extends ApiKey {
  owner_key: string;
  key_hash: string;
}

interface MockApiKeyStoreDependencies {
  now: () => Date;
  randomHex: (bytes: number) => string;
}

export class MockApiKeyStore {
  private nextId = 1;
  private records: MockApiKeyRecord[] = [];

  constructor(
    private readonly dependencies: MockApiKeyStoreDependencies = {
      now: () => new Date(),
      randomHex: bytes => randomBytes(bytes).toString('hex'),
    }
  ) {}

  list(actor: AuthUser, page: number, perPage: number): ApiKeyPage {
    const all = this.records
      .filter(record => record.owner_key === actor.id)
      .sort((left, right) => right.id - left.id)
      .map(publicApiKey);
    return paginated(all, page, perPage);
  }

  create(actor: AuthUser, input: CreateApiKeyInput): CreateApiKeyResult {
    const now = this.dependencies.now().toISOString();
    const keyPrefix = `luas_${this.dependencies.randomHex(6)}`;
    const plaintextKey = `${keyPrefix}.${this.dependencies.randomHex(24)}`;
    const record: MockApiKeyRecord = {
      id: this.nextId++,
      user_id: mockUserId(actor.id),
      owner_key: actor.id,
      name: input.name,
      key_prefix: keyPrefix,
      key_hash: createHash('sha256').update(plaintextKey).digest('hex'),
      scopes: [...new Set(input.scopes)].sort(),
      ...(input.expires_at ? { expires_at: input.expires_at } : {}),
      created_at: now,
      updated_at: now,
    };
    this.records = [record, ...this.records];
    return { api_key: publicApiKey(record), plaintext_key: plaintextKey };
  }

  revoke(actor: AuthUser, apiKeyId: number): boolean {
    const record = this.records.find(
      candidate => candidate.id === apiKeyId && candidate.owner_key === actor.id
    );
    if (!record) return false;
    if (!record.revoked_at) {
      const now = this.dependencies.now().toISOString();
      record.revoked_at = now;
      record.updated_at = now;
    }
    return true;
  }
}

export const mockApiKeyStore = new MockApiKeyStore();

function publicApiKey(record: MockApiKeyRecord): ApiKey {
  return {
    id: record.id,
    user_id: record.user_id,
    name: record.name,
    key_prefix: record.key_prefix,
    scopes: [...record.scopes],
    ...(record.last_used_at ? { last_used_at: record.last_used_at } : {}),
    ...(record.expires_at ? { expires_at: record.expires_at } : {}),
    ...(record.revoked_at ? { revoked_at: record.revoked_at } : {}),
    created_at: record.created_at,
    updated_at: record.updated_at,
  };
}

function mockUserId(ownerKey: string): number {
  const value = createHash('sha256').update(ownerKey).digest().readUIntBE(0, 6);
  return value || 1;
}

function paginated(items: ApiKey[], page: number, perPage: number): ApiKeyPage {
  const total = items.length;
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const currentPage = Math.min(page, lastPage);
  const start = (currentPage - 1) * perPage;
  const data = items.slice(start, start + perPage);
  const path = '/api/api-keys';
  const href = (target: number) => `${path}?page=${target}&per_page=${perPage}`;
  return {
    items: data,
    meta: {
      current_page: currentPage,
      per_page: perPage,
      total,
      last_page: lastPage,
      from: data.length ? start + 1 : 0,
      to: data.length ? start + data.length : 0,
    },
    links: {
      first: href(1),
      last: href(lastPage),
      prev: currentPage > 1 ? href(currentPage - 1) : null,
      next: currentPage < lastPage ? href(currentPage + 1) : null,
    },
  };
}
