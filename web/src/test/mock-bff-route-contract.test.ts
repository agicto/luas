import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const appApiRoot = resolve(process.cwd(), 'src/app/api');
const routeFileName = 'route.ts';
const routeHandlerPattern =
  /\bexport\s+(?:(?:async\s+)?function\s+(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s*\(|const\s+(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s*=)/g;
const unsafeMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

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

function productionGuard(path: string): string {
  return relativeRoute(path).startsWith('auth/')
    ? 'resolveAuthRoute('
    : 'guardMockBffRoute(';
}

interface RouteHandlerSource {
  method: string;
  source: string;
}

function routeHandlers(path: string): RouteHandlerSource[] {
  const source = readRoute(path);
  const matches = Array.from(source.matchAll(routeHandlerPattern));

  return matches.map((match, index) => ({
    method: match[1] ?? match[2],
    source: source.slice(match.index, matches[index + 1]?.index ?? source.length),
  }));
}

describe('mock BFF route contract', () => {
  const routeFiles = listRouteFiles(appApiRoot).sort((a, b) =>
    relativeRoute(a).localeCompare(relativeRoute(b))
  );

  it('discovers route handlers under src/app/api', () => {
    expect(routeFiles.length).toBeGreaterThan(0);
  });

  it('discovers at least one exported HTTP handler in every route file', () => {
    const offenders = routeFiles
      .filter((path) => routeHandlers(path).length === 0)
      .map(relativeRoute);

    expect(offenders).toEqual([]);
  });

  it('keeps every route handler behind its production availability guard', () => {
    const offenders = routeFiles.flatMap((path) =>
      routeHandlers(path)
        .filter((handler) => !handler.source.includes(productionGuard(path)))
        .map((handler) => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('keeps every unsafe mock BFF handler behind the same-origin guard', () => {
    const offenders = routeFiles.flatMap((path) =>
      routeHandlers(path)
        .filter((handler) => unsafeMethods.has(handler.method))
        .filter((handler) => {
          const availabilityGuard = handler.source.indexOf(productionGuard(path));
          const originGuard = handler.source.indexOf('guardSameOriginMutation(');

          return (
            originGuard < 0 ||
            availabilityGuard < 0 ||
            originGuard < availabilityGuard
          );
        })
        .map((handler) => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('keeps mock route JSON envelopes behind shared response helpers', () => {
    const offenders = routeFiles
      .filter((path) => /\bNextResponse\.json\s*\(/.test(readRoute(path)))
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
