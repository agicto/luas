import { MetadataRoute } from 'next';
import { env } from '@/config/env';

const BASE_URL = env.NEXT_PUBLIC_APP_URL;

export default function sitemap(): MetadataRoute.Sitemap {
  const routes = ['', '/login', '/register', '/console'].map((route) => ({
    url: `${BASE_URL}${route}`,
    lastModified: new Date(),
    changeFrequency: 'monthly' as const,
    priority: 0.8,
  }));

  return routes;
}
