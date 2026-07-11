import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');

function readSource(path: string): string {
  return readFileSync(resolve(sourceRoot, path), 'utf8');
}

describe('root runtime boundary', () => {
  it('uses Next.js error conventions instead of a custom root client wrapper', () => {
    const rootLayout = readSource('app/layout.tsx');
    const segmentErrorPath = resolve(sourceRoot, 'app/error.tsx');
    const globalErrorPath = resolve(sourceRoot, 'app/global-error.tsx');
    const legacyBoundaryPath = resolve(sourceRoot, 'components/error-boundary.tsx');

    expect(rootLayout).not.toContain('@/components/error-boundary');
    expect(existsSync(segmentErrorPath)).toBe(true);
    expect(existsSync(globalErrorPath)).toBe(true);
    expect(existsSync(legacyBoundaryPath)).toBe(false);

    const segmentError = readFileSync(segmentErrorPath, 'utf8');
    const globalError = readFileSync(globalErrorPath, 'utf8');

    expect(segmentError).toMatch(/^\s*['"]use client['"];?/);
    expect(segmentError).toContain('onClick={reset}');
    expect(globalError).toMatch(/^\s*['"]use client['"];?/);
    expect(globalError).toContain('<html');
    expect(globalError).toContain('<body');
    expect(globalError).toContain('onClick={reset}');
  });

  it('keeps toast rendering at the route groups that use it', () => {
    const rootLayout = readSource('app/layout.tsx');
    const authLayout = readSource('app/(auth)/layout.tsx');
    const protectedLayout = readSource('app/(protected)/layout.tsx');

    expect(rootLayout).not.toContain('@/components/ui/sonner');
    expect(authLayout).toContain('@/components/ui/sonner');
    expect(protectedLayout).toContain('@/components/ui/sonner');
  });

  it('keeps optional analytics out of a custom client boundary', () => {
    const analytics = readSource('components/analytics.tsx');

    expect(analytics).not.toMatch(/^\s*['"]use client['"];?/);
  });
});
