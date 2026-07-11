import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');

function readSource(path: string): string {
  return readFileSync(resolve(sourceRoot, path), 'utf8');
}

describe('i18n runtime boundary', () => {
  it('keeps the client translator free of server-only imports', () => {
    const clientTranslator = readSource('i18n/translations.ts');

    expect(clientTranslator.trimStart()).toMatch(/^['"]use client['"];?/);
    expect(clientTranslator).not.toContain('next-intl/server');
  });

  it('exposes server translations through a dedicated server entry', () => {
    const serverEntry = resolve(sourceRoot, 'i18n/server.ts');

    expect(existsSync(serverEntry)).toBe(true);
    if (!existsSync(serverEntry)) {
      return;
    }

    const serverTranslator = readFileSync(serverEntry, 'utf8');
    expect(serverTranslator).toContain('next-intl/server');
    expect(serverTranslator).toContain('getT');
    expect(serverTranslator).not.toMatch(/^\s*['"]use client['"];?/);
  });

  it('keeps the auth shell server-rendered', () => {
    const authLayout = readSource('app/(auth)/layout.tsx');

    expect(authLayout).not.toMatch(/^\s*['"]use client['"];?/);
    expect(authLayout).toContain("@/i18n/server");
  });
});
