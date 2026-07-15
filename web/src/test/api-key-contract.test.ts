import { describe, expect, it } from 'vitest';

import { MockApiKeyStore } from '@/features/api-key/server/mock-api-key-store';
import {
  parseApiKeyPageResponse,
  parseCreateApiKeyResponse,
} from '@/features/api-key/services/api-key-service';
import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

const timestamp = '2026-07-15T10:00:00Z';
const apiKey = {
  id: 42,
  user_id: 7,
  name: 'Deployment',
  key_prefix: 'luas_ab-cd_123456',
  scopes: ['models:invoke', 'models:read'],
  created_at: timestamp,
  updated_at: timestamp,
};

describe('API key browser contract', () => {
  it('parses token-free pages and one-time create results', () => {
    expect(parseApiKeyPageResponse(pageEnvelope([apiKey]))).toMatchObject({
      items: [apiKey],
      meta: { total: 1 },
    });
    expect(
      parseCreateApiKeyResponse({
        api_key: apiKey,
        plaintext_key: 'luas_ab-cd_123456.0123456789abcdef0123456789abcdef0123456789abcdef',
      })
    ).toMatchObject({ api_key: { id: 42 } });
  });

  it.each([
    { ...apiKey, key_hash: 'server-secret-hash' },
    { ...apiKey, plaintext_key: 'must-not-appear-in-list' },
    { ...apiKey, scopes: ['Models:Read'] },
    { ...apiKey, id: Number.MAX_SAFE_INTEGER + 1 },
  ])('rejects malformed or secret-bearing list metadata', value => {
    expectInvalidResponse(() => parseApiKeyPageResponse(pageEnvelope([value])));
  });

  it('rejects secret-bearing API key metadata inside create results', () => {
    expectInvalidResponse(() =>
      parseCreateApiKeyResponse({
        api_key: { ...apiKey, key_hash: 'server-secret-hash' },
        plaintext_key: 'luas_ab-cd_123456.0123456789abcdef0123456789abcdef0123456789abcdef',
      })
    );
  });

  it('keeps plaintext out of mock persistence and later list responses', () => {
    const store = new MockApiKeyStore({
      now: () => new Date(timestamp),
      randomHex: bytes => 'a'.repeat(bytes * 2),
    });
    const actor = { id: 'user-1', email: 'user@example.com', name: 'User' };
    const created = store.create(actor, {
      name: 'Deployment',
      scopes: ['models:read'],
    });

    expect(created.plaintext_key).toMatch(/^luas_[a-f0-9]+\.[a-f0-9]+$/);
    expect(JSON.stringify(store)).not.toContain(created.plaintext_key);
    expect(JSON.stringify(store.list(actor, 1, 15))).not.toContain(created.plaintext_key);
    expect(JSON.stringify(store.list(actor, 1, 15))).not.toContain('key_hash');
  });

  it('isolates mock keys by owner and makes revocation idempotent', () => {
    const store = new MockApiKeyStore({
      now: () => new Date(timestamp),
      randomHex: bytes => 'b'.repeat(bytes * 2),
    });
    const owner = { id: 'owner', email: 'owner@example.com', name: 'Owner' };
    const other = { id: 'other', email: 'other@example.com', name: 'Other' };
    const created = store.create(owner, { name: 'Owned', scopes: [] });

    expect(store.revoke(other, created.api_key.id)).toBe(false);
    expect(store.revoke(owner, created.api_key.id)).toBe(true);
    expect(store.revoke(owner, created.api_key.id)).toBe(true);
    expect(store.list(owner, 1, 15).items[0].revoked_at).toBe(
      new Date(timestamp).toISOString()
    );
    expect(store.list(other, 1, 15).items).toEqual([]);
  });
});

function pageEnvelope(data: unknown[]) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 15,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}

function expectInvalidResponse(operation: () => unknown): void {
  try {
    operation();
    throw new Error('Expected invalid response error');
  } catch (error) {
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).errorCode).toBe(ClientErrorCode.INVALID_RESPONSE);
  }
}
