/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_APP_NAME?: string;
  readonly VITE_API_BASE_URL?: string;
  readonly VITE_API_TIMEOUT_MS?: string;
  readonly VITE_API_MAX_RESPONSE_BYTES?: string;
  readonly VITE_DEFAULT_LOCALE?: 'en' | 'zh-CN';
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
