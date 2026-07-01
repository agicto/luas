import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const appApiRoot = resolve(process.cwd(), 'src/app/api');
const routeFileName = 'route.ts';

const forbiddenRoutePatterns = [
  {
    label: 'legacy ErrorCode constants',
    pattern: /\bErrorCode\b/,
  },
  {
    label: 'frontend-only ClientErrorCode',
    pattern: /\bClientErrorCode\b/,
  },
  {
    label: 'legacy underscore error code string',
    pattern: /['"`](?:SYS|AUTH|BIZ|VAL|NET|TIMEOUT|UNKNOWN)_\d{3}['"`]/,
  },
] as const;

function listRouteFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const path = resolve(dir, entry);
    const stat = statSync(path);

    if (stat.isDirectory()) {
      return listRouteFiles(path);
    }

    return entry === routeFileName ? [path] : [];
  });
}

function readRoute(path: string): string {
  return readFileSync(path, 'utf8');
}

function relativeRoute(path: string): string {
  return relative(appApiRoot, path);
}

describe('mock BFF route contract', () => {
  const routeFiles = listRouteFiles(appApiRoot).sort((a, b) =>
    relativeRoute(a).localeCompare(relativeRoute(b))
  );

  it('discovers route handlers under src/app/api', () => {
    expect(routeFiles.length).toBeGreaterThan(0);
  });

  it('keeps every mock BFF handler behind the production guard', () => {
    const offenders = routeFiles
      .filter((path) => !/\bguardMockBffRoute\s*\(/.test(readRoute(path)))
      .map(relativeRoute);

    expect(offenders).toEqual([]);
  });

  it('keeps mock route errors on canonical API error codes', () => {
    const offenders = routeFiles.flatMap((path) => {
      const source = readRoute(path);

      return forbiddenRoutePatterns
        .filter(({ pattern }) => pattern.test(source))
        .map(({ label }) => `${relativeRoute(path)}: ${label}`);
    });

    expect(offenders).toEqual([]);
  });
});
