export type AssetStatus = 'pending' | 'ready' | 'rejected';
export type AssetFilter = 'all' | AssetStatus;

export interface AssetItem {
  id: string;
  original_name: string;
  media_type: string;
  size_bytes: number;
  status: AssetStatus;
  created_at: string;
  ready_at: string | null;
}

export interface TransferGrant {
  method: 'GET' | 'PUT';
  url: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface UploadIntentResult {
  asset: AssetItem;
  upload: TransferGrant;
}

export interface AssetPage {
  items: AssetItem[];
  meta: {
    current_page: number;
    per_page: number;
    total: number;
    last_page: number;
    from: number;
    to: number;
  };
  links: {
    first: string;
    last: string;
    prev: string | null;
    next: string | null;
  };
}

export interface CreateUploadIntentInput {
  idempotency_key: string;
  original_name: string;
  media_type: string;
  size_bytes: number;
}

export interface DeleteAssetResult {
  deleted: true;
}
