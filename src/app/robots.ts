import { MetadataRoute } from 'next';
import { publicEnv } from '@/config/env';

const BASE_URL = publicEnv.NEXT_PUBLIC_APP_URL;

export default function robots(): MetadataRoute.Robots {

  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
        disallow: ['/api/', '/console/', '/admin/'],
      },
    ],
    sitemap: `${BASE_URL}/sitemap.xml`,
  };
}
