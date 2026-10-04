import type { z } from 'zod/mini';
import {
  assertContract,
  type Accepts,
  type ContractData,
  type ContractRequest,
  type ContractResponse,
  type Sends,
} from '@/http/contract';
import {
  assetPageEnvelopeSchema,
  assetSchema,
  createUploadIntentSchema,
  transferGrantSchema,
  uploadIntentResultSchema,
} from '@/features/asset/schemas';
import { downloadFromGrant, uploadToGrant } from '@/features/asset/services/asset-transfer';
import type {
  AssetFilter,
  AssetItem,
  AssetPage,
  CreateUploadIntentInput,
  TransferGrant,
  UploadIntentResult,
} from '@/features/asset/types';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import request, { ApiError } from '@/http/request';

interface ListOptions {
  page?: number;
  perPage?: number;
  status?: AssetFilter;
}

export const assetService = {
  async list({ page = 1, perPage = 25, status = 'all' }: ListOptions = {}): Promise<AssetPage> {
    const value = await request.getEnvelope<unknown>('/assets', {
      params: { page, per_page: perPage, status },
    });
    return parseAssetPage(value);
  },

  async createUploadIntent(input: CreateUploadIntentInput): Promise<UploadIntentResult> {
    return parseUploadIntent(
      await request.post<unknown, CreateUploadIntentInput>('/assets/upload-intents', input)
    );
  },

  async complete(assetId: string): Promise<AssetItem> {
    return parseAsset(await request.post<unknown>(`/assets/${assetId}/complete`));
  },

  async upload(file: File, idempotencyKey: string): Promise<AssetItem> {
    const input = createUploadIntentSchema.safeParse({
      idempotency_key: idempotencyKey,
      original_name: file.name,
      media_type: file.type,
      size_bytes: file.size,
    });
    if (!input.success) {
      throw new ApiError('The file is not an accepted asset', ApiErrorCode.COMMON_INVALID_INPUT);
    }
    const intent = await this.createUploadIntent(input.data);
    await uploadToGrant(intent.upload, file);
    return this.complete(intent.asset.id);
  },

  async download(
    asset: Pick<AssetItem, 'id' | 'original_name' | 'size_bytes' | 'media_type'>
  ): Promise<void> {
    const grant = parseTransferGrant(
      await request.post<unknown>(`/assets/${asset.id}/download-grant`)
    );
    await downloadFromGrant(grant, asset.original_name, asset.size_bytes, asset.media_type);
  },

  async delete(assetId: string): Promise<void> {
    await request.delete<void>(`/assets/${assetId}`);
  },
};

export function parseAssetPage(value: unknown): AssetPage {
  const parsed = assetPageEnvelopeSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return { items: parsed.data.data, meta: parsed.data.meta, links: parsed.data.links };
}

export function parseAsset(value: unknown): AssetItem {
  const parsed = assetSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseUploadIntent(value: unknown): UploadIntentResult {
  const parsed = uploadIntentResultSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

export function parseTransferGrant(value: unknown): TransferGrant {
  const parsed = transferGrantSchema.safeParse(value);
  if (!parsed.success) throw invalidResponse();
  return parsed.data;
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Asset service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}

assertContract<
  Accepts<z.input<typeof assetPageEnvelopeSchema>, ContractResponse<'listAssets', 200>>
>();
assertContract<
  Accepts<z.input<typeof uploadIntentResultSchema>, ContractData<'createAssetUploadIntent', 201>>
>();
assertContract<Sends<CreateUploadIntentInput, ContractRequest<'createAssetUploadIntent'>>>();
assertContract<Accepts<z.input<typeof assetSchema>, ContractData<'completeAssetUpload', 200>>>();
assertContract<
  Accepts<z.input<typeof transferGrantSchema>, ContractData<'createAssetDownloadGrant', 200>>
>();
