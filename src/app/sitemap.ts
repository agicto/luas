import { MetadataRoute } from 'next';
import { publicEnv } from '@/config/env';

const BASE_URL = publicEnv.NEXT_PUBLIC_APP_URL;

export default function sitemap(): MetadataRoute.Sitemap {

  const routes = [
    '',
    '/login',
    '/register',
  ];

  return routes.map((route) => ({
    url: `${BASE_URL}${route}`,
    lastModified: new Date(),
    changeFrequency: 'weekly' as const,
    priority: route === '' ? 1 : 0.8,
  }));
}
