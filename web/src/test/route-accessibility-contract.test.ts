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

  it('keeps a busy main landmark while protected routes initialize', () => {
    const authGuard = readSource(
      'features/auth/components/auth-guard.tsx'
    );

    expect(authGuard).toContain('aria-busy="true"');
  });

  it('keeps a main landmark in the styleguide devtool', () => {
    const styleguidePage = readSource(
      'app/(protected)/(devtools)/styleguide/page.tsx'
    );

    expect(styleguidePage.match(/<main\b/g) ?? []).toHaveLength(1);
    expect(styleguidePage.match(/<\/main>/g) ?? []).toHaveLength(1);
  });

  it('names icon-only loading examples while their icons are replaced', () => {
    const buttonShowcase = readSource(
      'components/features/styleguide/button-showcase.tsx'
    );

    expect(buttonShowcase).toContain('aria-label="Loading search"');
    expect(buttonShowcase).toContain('aria-label="Loading information"');
  });

  it('keeps public header links semantic and bounded on narrow screens', () => {
    const siteHeaderNav = readSource('components/features/site/site-header-nav.tsx');

    expect(siteHeaderNav).toContain("buttonVariants({ variant: 'ghost', size: 'sm' })");
    expect(siteHeaderNav).toContain("'interactive rounded-md max-sm:hidden'");
    expect(siteHeaderNav).not.toContain('<Button');
  });

  it('keeps stateful styleguide leaves behind an explicit client boundary', () => {
    const styleguidePage = readSource(
      'app/(protected)/(devtools)/styleguide/page.tsx'
    );
    const formShowcase = readSource(
      'components/features/styleguide/form-showcase.tsx'
    );

    expect(styleguidePage).not.toMatch(/^['"]use client['"]/);
    expect(formShowcase).toMatch(/^['"]use client['"]/);
    expect(formShowcase).toContain('React.useState');
  });
});
