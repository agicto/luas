import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, extname, relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');
const publicRouteRoots = [
  resolve(sourceRoot, 'app/layout.tsx'),
  resolve(sourceRoot, 'app/(site)'),
];
const sourceExtensions = ['.ts', '.tsx'] as const;

const forbiddenImports = [
  {
    label: 'auth feature runtime',
    matches: (specifier: string) => specifier === '@/features/auth' || specifier.startsWith('@/features/auth/'),
  },
  {
    label: 'authenticated providers',
    matches: (specifier: string) =>
      specifier === '@/providers' ||
      specifier === '@/providers/auth-provider' ||
      specifier === '@/providers/authenticated-providers' ||
      specifier === '@/providers/query-provider',
  },
  {
    label: 'React Query runtime',
    matches: (specifier: string) => specifier.startsWith('@tanstack/react-query'),
  },
  {
    label: 'HTTP client runtime',
    matches: (specifier: string) => specifier === '@/http' || specifier.startsWith('@/http/'),
  },
  {
    label: 'mock BFF route runtime',
    matches: (specifier: string) => specifier === '@/app/api' || specifier.startsWith('@/app/api/'),
  },
  {
    label: 'mock session runtime',
    matches: (specifier: string) =>
      specifier === '@/lib/session-signing' ||
      specifier === 'next/headers' ||
      specifier === 'next/cookies',
  },
  {
    label: 'Zustand store runtime',
    matches: (specifier: string) => specifier === 'zustand' || specifier.startsWith('zustand/'),
  },
  {
    label: 'toast runtime',
    matches: (specifier: string) =>
      specifier === 'sonner' || specifier === '@/components/ui/sonner',
  },
] as const;

const staticImportSpecifierPattern =
  /(?:import|export)\s+(?!type\b)(?:[\s\S]*?\s+from\s+)?['"]([^'"]+)['"]/g;
const dynamicImportSpecifierPattern = /import\(\s*['"]([^'"]+)['"]\s*\)/g;

function listSourceFiles(path: string): string[] {
  const stat = statSync(path);

  if (stat.isFile()) {
    return sourceExtensions.some((extension) => path.endsWith(extension)) ? [path] : [];
  }

  return readdirSync(path).flatMap((entry) => listSourceFiles(resolve(path, entry)));
}

function readImportSpecifiers(path: string): string[] {
  const source = readFileSync(path, 'utf8');

  return [
    ...Array.from(source.matchAll(staticImportSpecifierPattern), (match) => match[1]),
    ...Array.from(source.matchAll(dynamicImportSpecifierPattern), (match) => match[1]),
  ];
}

function resolveSourceImport(importer: string, specifier: string): string | null {
  if (!specifier.startsWith('.') && !specifier.startsWith('@/')) {
    return null;
  }

  const basePath = specifier.startsWith('@/')
    ? resolve(sourceRoot, specifier.slice(2))
    : resolve(dirname(importer), specifier);

  return resolveSourcePath(basePath);
}

function resolveSourcePath(path: string): string | null {
  if (existsSync(path) && statSync(path).isFile() && sourceExtensions.includes(extname(path) as never)) {
    return path;
  }

  for (const extension of sourceExtensions) {
    const filePath = `${path}${extension}`;

    if (existsSync(filePath) && statSync(filePath).isFile()) {
      return filePath;
    }
  }

  if (existsSync(path) && statSync(path).isDirectory()) {
    for (const extension of sourceExtensions) {
      const indexPath = resolve(path, `index${extension}`);

      if (existsSync(indexPath) && statSync(indexPath).isFile()) {
        return indexPath;
      }
    }
  }

  return null;
}

function relativeSource(path: string): string {
  return relative(sourceRoot, path);
}

describe('public route hydration boundary', () => {
  it('keeps public site routes free of auth, query, HTTP, mock BFF, mock-session, and toast runtime dependencies', () => {
    const entryFiles = publicRouteRoots.flatMap(listSourceFiles);
    const visited = new Set<string>();
    const offenders: string[] = [];

    function scan(path: string) {
      if (visited.has(path)) {
        return;
      }
      visited.add(path);

      for (const specifier of readImportSpecifiers(path)) {
        for (const { label, matches } of forbiddenImports) {
          if (matches(specifier)) {
            offenders.push(`${relativeSource(path)} imports ${specifier} (${label})`);
          }
        }

        const resolved = resolveSourceImport(path, specifier);
        if (resolved) {
          scan(resolved);
        }
      }
    }

    entryFiles.forEach(scan);

    expect(offenders.sort()).toEqual([]);
  });
});
