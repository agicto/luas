'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { assetService } from '@/features/asset/services/asset-service';
import type { AssetFilter, AssetItem } from '@/features/asset/types';

export const assetKeys = {
  all: ['assets'] as const,
  list: (status: AssetFilter) => [...assetKeys.all, 'list', status] as const,
};

export function useAssets(status: AssetFilter) {
  return useQuery({
    queryKey: assetKeys.list(status),
    queryFn: () => assetService.list({ status, page: 1, perPage: 100 }),
    staleTime: 15_000,
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useUploadAsset() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, idempotencyKey }: { file: File; idempotencyKey: string }) =>
      assetService.upload(file, idempotencyKey),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: assetKeys.all }),
  });
}

export function useDownloadAsset() {
  return useMutation({
    mutationFn: (asset: Pick<AssetItem, 'id' | 'original_name' | 'size_bytes' | 'media_type'>) =>
      assetService.download(asset),
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useDeleteAsset() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (assetId: string) => assetService.delete(assetId),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: assetKeys.all }),
  });
}
