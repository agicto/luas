import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');
const allowedProcessEnvFiles = new Set([
  'config/env.ts',
  'test/setup.ts',
]);

function listSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const path = resolve(dir, entry);
    const stat = statSync(path);

    if (stat.isDirectory()) {
      return listSourceFiles(path);
    }

    return path.endsWith('.ts') || path.endsWith('.tsx') ? [path] : [];
  });
}

function isAllowedProcessEnvFile(path: string): boolean {
  const relativePath = relative(sourceRoot, path);

  return (
    allowedProcessEnvFiles.has(relativePath) ||
    relativePath.endsWith('.test.ts') ||
    relativePath.endsWith('.test.tsx') ||
    relativePath.includes('__tests__/')
  );
}

describe('environment config contract', () => {
  it('keeps production source code behind src/config/env.ts', () => {
    const offenders = listSourceFiles(sourceRoot)
      .filter((path) => !isAllowedProcessEnvFile(path))
      .filter((path) => readFileSync(path, 'utf8').includes('process.env'))
      .map((path) => relative(sourceRoot, path));

    expect(offenders).toEqual([]);
  });
});
