export interface ApiKey {
  id: number;
  user_id: number;
  name: string;
  key_prefix: string;
  scopes: string[];
  last_used_at?: string;
  expires_at?: string;
  revoked_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiKeyPage {
  items: ApiKey[];
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

export interface CreateApiKeyInput {
  name: string;
  scopes: string[];
  expires_at?: string;
}

export interface CreateApiKeyResult {
  api_key: ApiKey;
  plaintext_key: string;
}
