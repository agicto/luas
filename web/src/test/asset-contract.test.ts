import { describe, expect, it, vi } from 'vitest';

import {
  parseAsset,
  parseAssetPage,
  parseTransferGrant,
  parseUploadIntent,
} from '@/features/asset/services/asset-service';
import { downloadFromGrant, safeTransferRequest } from '@/features/asset/services/asset-transfer';
import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

describe('asset browser contract', () => {
  it('parses only the public asset metadata envelope', () => {
    const page = parseAssetPage(pageEnvelope([asset()]));
    expect(page.items[0]).toMatchObject({ status: 'ready', size_bytes: 42 });
    expect(() => parseAsset({ ...asset(), object_key: 'private/key' })).toThrow(ApiError);
    expect(() => parseAsset({ ...asset(), checksum_sha256: 'secret' })).toThrow(ApiError);
  });

  it('rejects malformed grants and unknown intent fields', () => {
    expectInvalid(() =>
      parseTransferGrant({
        ...grant(),
        headers: { Authorization: 'Bearer secret' },
      })
    );
    expectInvalid(() =>
      parseUploadIntent({
        asset: asset(),
        upload: grant(),
        staging_key: 'private/key',
      })
    );
  });

  it('accepts HTTPS or exact same-origin transfer URLs and rejects unsafe requests', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-15T20:00:00Z'));
    const https = safeTransferRequest(grant());
    expect(https.url).toContain('https://storage.example/');

    const local = safeTransferRequest({
      ...grant(),
      url: `${window.location.origin}/api/asset-transfers/${'a'.repeat(32)}`,
    });
    expect(local.url).toContain('/api/asset-transfers/');

    expectInvalid(() => safeTransferRequest({ ...grant(), url: 'http://storage.example/upload' }));
    expectInvalid(() =>
      safeTransferRequest({ ...grant(), url: 'https://user:pass@storage.example/upload' })
    );
    expectInvalid(() =>
      safeTransferRequest({
        ...grant(),
        headers: { authorization: 'Bearer secret' },
      })
    );
    vi.useRealTimers();
  });

  it('rejects download bytes or media metadata that differ from the asset record', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-15T20:00:00Z'));
    const fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal('fetch', fetchMock);
    const downloadGrant = { ...grant(), method: 'GET' as const };

    fetchMock.mockResolvedValueOnce(
      new Response(new Uint8Array([1, 2, 3]), {
        headers: { 'content-length': '3', 'content-type': 'application/pdf' },
      })
    );
    await expect(
      downloadFromGrant(downloadGrant, 'report.pdf', 2, 'application/pdf')
    ).rejects.toMatchObject({ errorCode: ClientErrorCode.INVALID_RESPONSE });

    fetchMock.mockResolvedValueOnce(
      new Response(new Uint8Array([1, 2]), {
        headers: { 'content-type': 'text/plain' },
      })
    );
    await expect(
      downloadFromGrant(downloadGrant, 'report.pdf', 2, 'application/pdf')
    ).rejects.toMatchObject({ errorCode: ClientErrorCode.INVALID_RESPONSE });

    vi.unstubAllGlobals();
    vi.useRealTimers();
  });
});

function expectInvalid(call: () => unknown): void {
  try {
    call();
    throw new Error('expected invalid response');
  } catch (error) {
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).errorCode).toBe(ClientErrorCode.INVALID_RESPONSE);
  }
}

function asset() {
  return {
    id: '019bf6d8-17c5-7a98-a084-6d45793f5f0c',
    original_name: 'report.pdf',
    media_type: 'application/pdf',
    size_bytes: 42,
    status: 'ready' as const,
    created_at: '2026-07-15T20:00:00Z',
    ready_at: '2026-07-15T20:00:04Z',
  };
}

function grant() {
  return {
    method: 'PUT' as const,
    url: 'https://storage.example/upload?signature=secret',
    headers: { 'content-type': 'application/pdf' },
    expires_at: '2026-07-15T20:10:00Z',
  };
}

function pageEnvelope(data: unknown[]) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 25,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}
