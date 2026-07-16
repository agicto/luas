import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function source(relativePath: string): string {
  return readFileSync(resolve(process.cwd(), relativePath), 'utf8');
}

describe('console public boundary', () => {
  it('uses real starter routes without coupling the console to a source host', () => {
    const page = source('src/app/(protected)/(console)/console/page.tsx');
    const english = source('src/i18n/modules/console/en-US.ts');
    const chinese = source('src/i18n/modules/console/zh-Hans.ts');

    expect(page).toContain("t('home.quickLinks.apiAccess.title')");
    expect(page).toContain('href: ROUTES.CONSOLE.SETTINGS');
    expect(page).toContain('href={ROUTES.SITE.HOME}');
    expect(`${page}\n${english}\n${chinese}`).not.toMatch(/github\.com|OpenAPI/i);
    expect(english).toContain('Manage user-owned API keys');
    expect(chinese).toContain('管理用户 API 密钥');
  });
});
