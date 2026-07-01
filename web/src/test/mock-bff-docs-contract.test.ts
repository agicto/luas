import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const mockBffGuidePath = resolve(process.cwd(), 'docs/MOCK_BFF.md');
const readmePath = resolve(process.cwd(), 'README.md');

function read(path: string): string {
  return readFileSync(path, 'utf8');
}

describe('mock BFF documentation contract', () => {
  it('keeps the downstream replacement guide discoverable', () => {
    expect(read(readmePath)).toContain('docs/MOCK_BFF.md');
  });

  it('keeps the replacement guide tied to the runtime and contract seams', () => {
    const guide = read(mockBffGuidePath);

    expect(guide).toContain('# Mock BFF Replacement Guide');
    expect(guide).toContain('NEXT_PUBLIC_API_URL');
    expect(guide).toContain('MOCK_BFF_ENABLED');
    expect(guide).toContain('guardMockBffRoute()');
    expect(guide).toContain('../../contracts/README.md');
    expect(guide).toContain('src/test/mock-bff-route-contract.test.ts');
    expect(guide).toContain('src/test/error-code-vocabulary.test.ts');
  });
});
