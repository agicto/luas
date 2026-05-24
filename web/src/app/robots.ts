import { MetadataRoute } from 'next';
import { env } from '@/config/env';

const BASE_URL = env.NEXT_PUBLIC_APP_URL;

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: '*', allow: '/' }],
    sitemap: `${BASE_URL}/sitemap.xml`,
  };
}
