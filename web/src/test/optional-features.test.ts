import { describe, expect, it } from 'vitest';

import { parseOptionalWebFeatures } from '@/config/optional-features';

describe('optional Web feature selection', () => {
  it('keeps the default Web shell free of optional features', () => {
    expect(parseOptionalWebFeatures('')).toEqual([]);
  });

  it('selects the organization feature by its canonical name', () => {
    expect(parseOptionalWebFeatures('organization')).toEqual(['organization']);
    expect(parseOptionalWebFeatures('permission,organization')).toEqual([
      'permission',
      'organization',
    ]);
  });

  it.each([' organization', 'organization ', 'Organization', 'organization,organization', 'billing'])(
    'rejects ambiguous or unknown selection %s',
    (selection) => {
      expect(() => parseOptionalWebFeatures(selection)).toThrow();
    }
  );

  it('requires feature dependencies explicitly', () => {
    expect(() => parseOptionalWebFeatures('permission')).toThrow(
      'requires "organization"'
    );
  });
});
