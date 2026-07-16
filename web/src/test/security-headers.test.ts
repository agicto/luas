import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

import { getBrowserSecurityHeaders } from '../../security-headers';

function asRecord(production: boolean): Record<string, string> {
  return Object.fromEntries(
    getBrowserSecurityHeaders({ production }).map(header => [header.key, header.value])
  );
}

describe('browser security response policy', () => {
  it('emits unique, injection-safe common headers', () => {
    const headers = getBrowserSecurityHeaders({ production: false });
    const normalizedKeys = headers.map(header => header.key.toLowerCase());

    expect(new Set(normalizedKeys).size).toBe(headers.length);
    expect(headers).toHaveLength(8);
    for (const header of headers) {
      expect(header.key).not.toMatch(/[\r\n]/);
      expect(header.value).not.toMatch(/[\r\n]/);
    }
  });

  it('uses a structural CSP without weakening script execution policy', () => {
    const csp = asRecord(false)['Content-Security-Policy'];
    const directives = csp.split('; ').sort();

    expect(directives).toEqual(
      [
        "base-uri 'self'",
        "form-action 'self'",
        "frame-ancestors 'none'",
        "object-src 'none'",
      ].sort()
    );
    expect(csp).not.toMatch(/(?:default|script|style|connect|img|font)-src/);
    expect(csp).not.toContain("'unsafe-");
  });

  it('denies framing and unused browser capabilities by default', () => {
    const headers = asRecord(false);

    expect(headers['X-Frame-Options']).toBe('DENY');
    expect(headers['X-XSS-Protection']).toBe('0');
    expect(headers['X-Content-Type-Options']).toBe('nosniff');
    expect(headers['Permissions-Policy']).toBe(
      'browsing-topics=(), camera=(), geolocation=(), microphone=(), payment=(), usb=()'
    );
  });

  it('adds host-only HSTS in production without claiming subdomain or preload ownership', () => {
    expect(asRecord(false)).not.toHaveProperty('Strict-Transport-Security');
    expect(asRecord(true)['Strict-Transport-Security']).toBe('max-age=31536000');
    expect(asRecord(true)['Strict-Transport-Security']).not.toMatch(/includeSubDomains|preload/i);
  });

  it('wires the policy through Next config and uses the Next 16 proxy convention', () => {
    const root = process.cwd();
    const nextConfig = readFileSync(resolve(root, 'next.config.ts'), 'utf8');
    const proxy = readFileSync(resolve(root, 'src/proxy.ts'), 'utf8');

    expect(nextConfig).toContain("from './security-headers'");
    expect(nextConfig).toContain('getBrowserSecurityHeaders({');
    expect(nextConfig).toContain("production: process.env.NODE_ENV === 'production'");
    expect(existsSync(resolve(root, 'middleware.ts'))).toBe(false);
    expect(existsSync(resolve(root, 'proxy.ts'))).toBe(false);
    expect(proxy).toContain('export async function proxy(');
    expect(proxy).not.toContain('export async function middleware(');
  });
});
