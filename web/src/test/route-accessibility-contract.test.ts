import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');

function readSource(path: string): string {
  return readFileSync(resolve(sourceRoot, path), 'utf8');
}

describe('route accessibility contracts', () => {
  it('keeps a main landmark in the public auth shell', () => {
    const authLayout = readSource('app/(auth)/layout.tsx');

    expect(authLayout.match(/<main\b/g) ?? []).toHaveLength(1);
    expect(authLayout.match(/<\/main>/g) ?? []).toHaveLength(1);
  });
});
