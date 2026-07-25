import { z } from 'zod';

const publicPathOrUrl = z
  .string()
  .min(1)
  .refine(
    (value) => {
      if (value.startsWith('/') && !value.startsWith('//')) {
        return true;
      }
      try {
        const url = new URL(value);
        return (
          url.protocol === 'https:' || (url.protocol === 'http:' && url.hostname === 'localhost')
        );
      } catch {
        return false;
      }
    },
    {
      message: 'must be a root-relative path, HTTPS URL, or localhost HTTP URL',
    },
  );

const integerFromString = (fallback: number, minimum: number, maximum: number) =>
  z.preprocess(
    (value) => (value === undefined || value === '' ? fallback : Number(value)),
    z.number().int().min(minimum).max(maximum),
  );

const envSchema = z.object({
  APP_NAME: z.string().trim().min(1).max(60),
  API_BASE_URL: publicPathOrUrl,
  API_TIMEOUT_MS: integerFromString(8_000, 1_000, 30_000),
  API_MAX_RESPONSE_BYTES: integerFromString(1_048_576, 16_384, 5_242_880),
  DEFAULT_LOCALE: z.enum(['en', 'zh-CN']),
});

const parsed = envSchema.safeParse({
  APP_NAME: import.meta.env.VITE_APP_NAME || 'Luas',
  API_BASE_URL: import.meta.env.VITE_API_BASE_URL || '/api',
  API_TIMEOUT_MS: import.meta.env.VITE_API_TIMEOUT_MS,
  API_MAX_RESPONSE_BYTES: import.meta.env.VITE_API_MAX_RESPONSE_BYTES,
  DEFAULT_LOCALE: import.meta.env.VITE_DEFAULT_LOCALE || 'en',
});

if (!parsed.success) {
  const fields = parsed.error.issues.map((issue) => issue.path.join('.')).join(', ');
  throw new Error(`Invalid public SPA environment: ${fields}`);
}

export const env = Object.freeze(parsed.data);
