import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

import { getBrowserSecurityHeaders } from './security-headers';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  output: 'standalone',

  // Performance optimizations
  compress: true,
  poweredByHeader: false,

  // Image optimization
  images: {
    formats: ['image/avif', 'image/webp'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
  },

  // Experimental features
  experimental: {
    optimizePackageImports: ['lucide-react'],
  },

  productionBrowserSourceMaps: false,
  turbopack: {},

  // Browser response policy only. Cache ownership remains route-specific.
  async headers() {
    return [
      {
        source: '/:path*',
        headers: getBrowserSecurityHeaders({
          production: process.env.NODE_ENV === 'production',
        }),
      },
    ];
  },
};

export default withNextIntl(nextConfig);
