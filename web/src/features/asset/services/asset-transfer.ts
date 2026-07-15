import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import type { TransferGrant } from '@/features/asset/types';
import { ASSET_MAX_BROWSER_BYTES } from '@/features/asset/schemas';

const localTransferPath = /^\/(?:api|v1)\/asset-transfers\/[A-Za-z0-9._-]{32,256}$/;
const forbiddenHeaders = new Set([
  'authorization',
  'cookie',
  'content-length',
  'host',
  'origin',
  'referer',
  'set-cookie',
]);

export async function uploadToGrant(grant: TransferGrant, file: File): Promise<void> {
  if (grant.method !== 'PUT') throw invalidGrant();
  const request = safeTransferRequest(grant);
  const response = await fetch(request.url, {
    method: 'PUT',
    headers: request.headers,
    body: file,
    credentials: 'omit',
    cache: 'no-store',
    redirect: 'error',
    referrerPolicy: 'no-referrer',
  });
  if (!response.ok) {
    await response.body?.cancel();
    throw transferFailed(response.status);
  }
  await response.body?.cancel();
}

export async function downloadFromGrant(
  grant: TransferGrant,
  downloadName: string,
  expectedBytes: number,
  mediaType: string
): Promise<void> {
  if (grant.method !== 'GET') throw invalidGrant();
  if (
    !Number.isSafeInteger(expectedBytes) ||
    expectedBytes <= 0 ||
    expectedBytes > ASSET_MAX_BROWSER_BYTES
  ) {
    throw invalidGrant();
  }
  const request = safeTransferRequest(grant);
  const response = await fetch(request.url, {
    method: 'GET',
    headers: request.headers,
    credentials: 'omit',
    cache: 'no-store',
    redirect: 'error',
    referrerPolicy: 'no-referrer',
  });
  if (!response.ok) {
    await response.body?.cancel();
    throw transferFailed(response.status);
  }
  const declaredLength = response.headers.get('content-length');
  if (
    declaredLength !== null &&
    (!/^\d+$/.test(declaredLength) || Number(declaredLength) !== expectedBytes)
  ) {
    await response.body?.cancel();
    throw invalidDownload();
  }
  const declaredMediaType = response.headers.get('content-type')?.split(';', 1)[0]?.trim();
  if (declaredMediaType && declaredMediaType.toLowerCase() !== mediaType.toLowerCase()) {
    await response.body?.cancel();
    throw invalidDownload();
  }
  const blob = await readBoundedBlob(response, expectedBytes, mediaType);
  const objectUrl = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement('a');
    anchor.href = objectUrl;
    anchor.download = downloadName;
    anchor.rel = 'noopener noreferrer';
    anchor.click();
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}

async function readBoundedBlob(
  response: Response,
  expectedBytes: number,
  mediaType: string
): Promise<Blob> {
  if (!response.body) throw invalidDownload();
  const reader = response.body.getReader();
  const chunks: ArrayBuffer[] = [];
  let received = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      received += value.byteLength;
      if (received > expectedBytes) {
        await reader.cancel();
        throw invalidDownload();
      }
      chunks.push(Uint8Array.from(value).buffer);
    }
  } finally {
    reader.releaseLock();
  }
  if (received !== expectedBytes) throw invalidDownload();
  return new Blob(chunks, { type: mediaType });
}

export function safeTransferRequest(grant: TransferGrant): {
  url: string;
  headers: Headers;
} {
  if (new Date(grant.expires_at).getTime() <= Date.now()) throw invalidGrant();
  let url: URL;
  try {
    url = new URL(grant.url, window.location.origin);
  } catch {
    throw invalidGrant();
  }
  if (url.username || url.password || url.hash) throw invalidGrant();
  const sameOriginLocal =
    url.origin === window.location.origin && localTransferPath.test(url.pathname);
  if (url.protocol !== 'https:' && !sameOriginLocal) throw invalidGrant();

  const headers = new Headers();
  for (const [name, value] of Object.entries(grant.headers)) {
    const normalized = name.toLowerCase();
    if (name !== normalized || forbiddenHeaders.has(normalized)) throw invalidGrant();
    headers.set(normalized, value);
  }
  return { url: url.toString(), headers };
}

function invalidGrant(): ApiError {
  return new ApiError('Asset transfer grant is invalid', ClientErrorCode.INVALID_RESPONSE);
}

function invalidDownload(): ApiError {
  return new ApiError(
    'Asset download did not match its metadata',
    ClientErrorCode.INVALID_RESPONSE
  );
}

function transferFailed(status: number): ApiError {
  return new ApiError('Asset transfer failed', ClientErrorCode.FETCH_ERROR, status);
}
