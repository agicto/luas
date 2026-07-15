import 'server-only';

import { createHash } from 'node:crypto';

import type { ApiPaginationLinks, ApiPaginationMeta } from '@/app/api/_shared/success-response';
import { ASSET_DEFAULT_UPLOAD_BYTES, extensionMatchesMediaType } from '@/features/asset/schemas';
import type { AuthUser } from '@/features/auth/types';
import type {
  AssetFilter,
  AssetItem,
  CreateUploadIntentInput,
  TransferGrant,
  UploadIntentResult,
} from '@/features/asset/types';
import { ApiErrorCode, type ApiErrorCodeValue } from '@/http/codes';

const uploadGrantMs = 10 * 60 * 1_000;
const downloadGrantMs = 5 * 60 * 1_000;
const pendingLifetimeMs = 60 * 60 * 1_000;
const maxAssetsPerUser = 100;
const maxMockBytesPerUser = 64 * 1_024 * 1_024;
const maxTransferTokens = 2_048;

interface MockAssetRecord {
  item: AssetItem;
  idempotencyKey: string;
  requestHash: string;
  pendingExpiresAt: number;
  staging?: Uint8Array;
  object?: Uint8Array;
  checksum?: string;
  deleted: boolean;
}

interface MockAssetState {
  assets: MockAssetRecord[];
}

interface MockTransferToken {
  operation: 'download' | 'upload';
  assetId: string;
  userId: string;
  expiresAt: number;
}

export class MockAssetError extends Error {
  constructor(
    readonly status: number,
    readonly errorCode: ApiErrorCodeValue,
    message: string
  ) {
    super(message);
    this.name = 'MockAssetError';
  }
}

const states = new Map<string, MockAssetState>();
const transferTokens = new Map<string, MockTransferToken>();

export const mockAssetStore = {
  list(
    user: AuthUser,
    page: number,
    perPage: number,
    status: AssetFilter
  ): { items: AssetItem[]; meta: ApiPaginationMeta; links: ApiPaginationLinks } {
    const matching = stateFor(user)
      .assets.filter(
        record => !record.deleted && (status === 'all' || record.item.status === status)
      )
      .sort((left, right) => right.item.created_at.localeCompare(left.item.created_at));
    const start = (page - 1) * perPage;
    const items = matching.slice(start, start + perPage).map(record => cloneAsset(record.item));
    return {
      items,
      meta: pageMeta(matching.length, page, perPage, items.length),
      links: pageLinks(matching.length, page, perPage, status),
    };
  },

  createUploadIntent(
    user: AuthUser,
    input: CreateUploadIntentInput,
    origin: string
  ): UploadIntentResult {
    if (!extensionMatchesMediaType(input.original_name, input.media_type)) {
      throw invalidMediaType();
    }
    if (input.size_bytes > ASSET_DEFAULT_UPLOAD_BYTES) throw sizeExceeded();
    const state = stateFor(user);
    const fingerprint = requestFingerprint(input);
    const existing = state.assets.find(record => record.idempotencyKey === input.idempotency_key);
    if (existing) {
      if (existing.requestHash !== fingerprint) {
        throw new MockAssetError(
          409,
          ApiErrorCode.ASSET_IDEMPOTENCY_CONFLICT,
          'Asset idempotency key conflicts with an existing request'
        );
      }
      if (existing.deleted || existing.item.status !== 'pending') throw notReady();
      if (existing.pendingExpiresAt <= Date.now()) throw uploadExpired();
      return {
        asset: cloneAsset(existing.item),
        upload: issueGrant(user.id, existing.item.id, 'upload', origin),
      };
    }
    if (state.assets.filter(record => !record.deleted).length >= maxAssetsPerUser) {
      throw unavailable();
    }
    const now = new Date();
    const record: MockAssetRecord = {
      item: {
        id: crypto.randomUUID(),
        original_name: input.original_name,
        media_type: input.media_type,
        size_bytes: input.size_bytes,
        status: 'pending',
        created_at: now.toISOString(),
        ready_at: null,
      },
      idempotencyKey: input.idempotency_key,
      requestHash: fingerprint,
      pendingExpiresAt: now.getTime() + pendingLifetimeMs,
      deleted: false,
    };
    state.assets.push(record);
    return {
      asset: cloneAsset(record.item),
      upload: issueGrant(user.id, record.item.id, 'upload', origin),
    };
  },

  async acceptUpload(token: string, request: Request): Promise<void> {
    const transfer = resolveToken(token, 'upload');
    const record = recordForTransfer(transfer);
    if (record.deleted || record.item.status !== 'pending') throw notReady();
    if (record.pendingExpiresAt <= Date.now()) throw uploadExpired();
    if (request.headers.get('content-type') !== record.item.media_type) throw invalidMediaType();
    const declaredLength = request.headers.get('content-length');
    if (
      declaredLength &&
      (!/^\d+$/.test(declaredLength) || Number(declaredLength) !== record.item.size_bytes)
    ) {
      throw sizeExceeded();
    }
    const bytes = await readExactBody(request.body, record.item.size_bytes);
    if (
      storedBytesForUser(transfer.userId, record.item.id) + bytes.byteLength >
      maxMockBytesPerUser
    ) {
      throw unavailable();
    }
    record.staging = Uint8Array.from(bytes);
  },

  complete(user: AuthUser, assetId: string): AssetItem {
    const record = ownedRecord(user, assetId);
    if (record.item.status === 'ready') return cloneAsset(record.item);
    if (record.item.status !== 'pending') throw notReady();
    if (record.pendingExpiresAt <= Date.now()) throw uploadExpired();
    if (!record.staging || record.staging.byteLength !== record.item.size_bytes) {
      rejectRecord(record);
      throw notReady();
    }
    if (!contentMatches(record.item.media_type, record.staging)) {
      rejectRecord(record);
      throw invalidMediaType();
    }
    record.object = Uint8Array.from(record.staging);
    record.staging = undefined;
    record.checksum = createHash('sha256').update(record.object).digest('hex');
    record.item.status = 'ready';
    record.item.ready_at = new Date().toISOString();
    return cloneAsset(record.item);
  },

  downloadGrant(user: AuthUser, assetId: string, origin: string): TransferGrant {
    const record = ownedRecord(user, assetId);
    if (record.item.status !== 'ready' || !record.object) throw notReady();
    return issueGrant(user.id, assetId, 'download', origin);
  },

  download(token: string): Response {
    const transfer = resolveToken(token, 'download');
    const record = recordForTransfer(transfer);
    if (record.deleted) throw notFound();
    if (record.item.status !== 'ready' || !record.object) throw notReady();
    return new Response(Buffer.from(record.object), {
      status: 200,
      headers: {
        'cache-control': 'private, no-store',
        'content-disposition': contentDisposition(record.item.original_name),
        'content-length': String(record.object.byteLength),
        'content-type': record.item.media_type,
      },
    });
  },

  delete(user: AuthUser, assetId: string): boolean {
    const record = ownedRecord(user, assetId, true);
    if (record.deleted) return true;
    record.staging = undefined;
    record.object = undefined;
    record.checksum = undefined;
    record.deleted = true;
    return true;
  },

  reset(): void {
    states.clear();
    transferTokens.clear();
  },
};

function stateFor(user: AuthUser): MockAssetState {
  const existing = states.get(user.id);
  if (existing) return existing;
  const state: MockAssetState = { assets: [] };
  states.set(user.id, state);
  return state;
}

function ownedRecord(user: AuthUser, assetId: string, includeDeleted = false): MockAssetRecord {
  const record = stateFor(user).assets.find(candidate => candidate.item.id === assetId);
  if (!record || (!includeDeleted && record.deleted)) throw notFound();
  return record;
}

function recordForTransfer(transfer: MockTransferToken): MockAssetRecord {
  const state = states.get(transfer.userId);
  const record = state?.assets.find(candidate => candidate.item.id === transfer.assetId);
  if (!record) throw notFound();
  return record;
}

function issueGrant(
  userId: string,
  assetId: string,
  operation: MockTransferToken['operation'],
  origin: string
): TransferGrant {
  pruneExpiredTokens();
  if (transferTokens.size >= maxTransferTokens) throw unavailable();
  const token = `${crypto.randomUUID().replaceAll('-', '')}.${crypto.randomUUID().replaceAll('-', '')}`;
  const ttl = operation === 'upload' ? uploadGrantMs : downloadGrantMs;
  const expiresAt = Date.now() + ttl;
  transferTokens.set(token, { operation, assetId, userId, expiresAt });
  return {
    method: operation === 'upload' ? 'PUT' : 'GET',
    url: `${origin}/api/asset-transfers/${token}`,
    headers:
      operation === 'upload'
        ? {
            'content-type': recordForTransfer({ operation, assetId, userId, expiresAt }).item
              .media_type,
          }
        : {},
    expires_at: new Date(expiresAt).toISOString(),
  };
}

function resolveToken(token: string, operation: MockTransferToken['operation']): MockTransferToken {
  const transfer = transferTokens.get(token);
  if (!transfer || transfer.operation !== operation || transfer.expiresAt <= Date.now()) {
    transferTokens.delete(token);
    throw notFound();
  }
  return transfer;
}

function pruneExpiredTokens(): void {
  const now = Date.now();
  for (const [token, transfer] of transferTokens) {
    if (transfer.expiresAt <= now) transferTokens.delete(token);
  }
}

function requestFingerprint(input: CreateUploadIntentInput): string {
  return createHash('sha256')
    .update(
      JSON.stringify({
        original_name: input.original_name,
        media_type: input.media_type,
        size_bytes: input.size_bytes,
      })
    )
    .digest('hex');
}

function storedBytesForUser(userId: string, replacingAssetId: string): number {
  return (states.get(userId)?.assets ?? []).reduce((total, record) => {
    if (record.item.id === replacingAssetId || record.deleted) return total;
    return total + (record.staging?.byteLength ?? 0) + (record.object?.byteLength ?? 0);
  }, 0);
}

async function readExactBody(
  body: ReadableStream<Uint8Array> | null,
  expectedBytes: number
): Promise<Uint8Array> {
  if (!body || expectedBytes <= 0 || expectedBytes > ASSET_DEFAULT_UPLOAD_BYTES)
    throw sizeExceeded();
  const output = new Uint8Array(expectedBytes);
  const reader = body.getReader();
  let offset = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      if (offset + value.byteLength > expectedBytes) throw sizeExceeded();
      output.set(value, offset);
      offset += value.byteLength;
    }
  } catch (error) {
    await reader.cancel().catch(() => undefined);
    throw error;
  } finally {
    reader.releaseLock();
  }
  if (offset !== expectedBytes) throw sizeExceeded();
  return output;
}

function contentMatches(mediaType: string, bytes: Uint8Array): boolean {
  if (mediaType === 'image/jpeg') {
    return bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff;
  }
  if (mediaType === 'image/png') {
    const signature = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];
    return (
      bytes.length >= signature.length && signature.every((value, index) => bytes[index] === value)
    );
  }
  if (mediaType === 'image/webp') {
    return bytes.length >= 12 && ascii(bytes, 0, 4) === 'RIFF' && ascii(bytes, 8, 12) === 'WEBP';
  }
  if (mediaType === 'application/pdf') {
    return bytes.length >= 5 && ascii(bytes, 0, 5) === '%PDF-';
  }
  if (mediaType === 'text/plain' || mediaType === 'text/csv') {
    if (bytes.includes(0)) return false;
    try {
      new TextDecoder('utf-8', { fatal: true }).decode(bytes);
      return true;
    } catch {
      return false;
    }
  }
  return false;
}

function ascii(bytes: Uint8Array, start: number, end: number): string {
  return String.fromCharCode(...bytes.slice(start, end));
}

function rejectRecord(record: MockAssetRecord): void {
  record.staging = undefined;
  record.object = undefined;
  record.item.status = 'rejected';
  record.item.ready_at = null;
}

function cloneAsset(item: AssetItem): AssetItem {
  return { ...item };
}

function pageMeta(
  total: number,
  page: number,
  perPage: number,
  itemCount: number
): ApiPaginationMeta {
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const from = total === 0 ? 0 : (page - 1) * perPage + 1;
  return {
    current_page: page,
    per_page: perPage,
    total,
    last_page: lastPage,
    from,
    to: itemCount === 0 ? 0 : from + itemCount - 1,
  };
}

function pageLinks(
  total: number,
  page: number,
  perPage: number,
  status: AssetFilter
): ApiPaginationLinks {
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const href = (target: number) =>
    `/api/assets?page=${target}&per_page=${perPage}&status=${status}`;
  return {
    first: href(1),
    last: href(lastPage),
    prev: page > 1 ? href(page - 1) : null,
    next: page < lastPage ? href(page + 1) : null,
  };
}

function contentDisposition(filename: string): string {
  return `attachment; filename*=UTF-8''${encodeURIComponent(filename).replaceAll("'", '%27')}`;
}

function notFound(): MockAssetError {
  return new MockAssetError(404, ApiErrorCode.ASSET_NOT_FOUND, 'Asset not found');
}

function notReady(): MockAssetError {
  return new MockAssetError(409, ApiErrorCode.ASSET_NOT_READY, 'Asset is not ready');
}

function uploadExpired(): MockAssetError {
  return new MockAssetError(410, ApiErrorCode.ASSET_UPLOAD_EXPIRED, 'Asset upload expired');
}

function sizeExceeded(): MockAssetError {
  return new MockAssetError(413, ApiErrorCode.ASSET_SIZE_EXCEEDED, 'Asset size is invalid');
}

function invalidMediaType(): MockAssetError {
  return new MockAssetError(
    422,
    ApiErrorCode.ASSET_INVALID_MEDIA_TYPE,
    'Asset media type is invalid'
  );
}

function unavailable(): MockAssetError {
  return new MockAssetError(
    503,
    ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
    'Mock asset store is unavailable'
  );
}
