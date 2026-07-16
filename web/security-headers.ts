export interface BrowserSecurityHeader {
  key: string;
  value: string;
}

const structuralContentSecurityPolicy = [
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
  "object-src 'none'",
].join('; ');

const permissionsPolicy = [
  'browsing-topics=()',
  'camera=()',
  'geolocation=()',
  'microphone=()',
  'payment=()',
  'usb=()',
].join(', ');

const commonHeaders: readonly BrowserSecurityHeader[] = [
  {
    key: 'Content-Security-Policy',
    value: structuralContentSecurityPolicy,
  },
  {
    key: 'Permissions-Policy',
    value: permissionsPolicy,
  },
  {
    key: 'Referrer-Policy',
    value: 'strict-origin-when-cross-origin',
  },
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff',
  },
  {
    key: 'X-DNS-Prefetch-Control',
    value: 'on',
  },
  {
    key: 'X-Frame-Options',
    value: 'DENY',
  },
  {
    key: 'X-Permitted-Cross-Domain-Policies',
    value: 'none',
  },
  {
    key: 'X-XSS-Protection',
    value: '0',
  },
];

const productionHeaders: readonly BrowserSecurityHeader[] = [
  {
    key: 'Strict-Transport-Security',
    value: 'max-age=31536000',
  },
];

export function getBrowserSecurityHeaders(options: {
  production: boolean;
}): BrowserSecurityHeader[] {
  const headers = options.production
    ? [...commonHeaders, ...productionHeaders]
    : [...commonHeaders];

  return headers.map(header => ({ ...header }));
}
