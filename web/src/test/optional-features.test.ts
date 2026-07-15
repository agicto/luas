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

  it('selects asset independently', () => {
    expect(parseOptionalWebFeatures('asset')).toEqual(['asset']);
  });

  it('selects setting only with organization ownership', () => {
    expect(parseOptionalWebFeatures('organization,setting')).toEqual([
      'organization',
      'setting',
    ]);
    expect(() => parseOptionalWebFeatures('setting')).toThrow('requires "organization"');
  });

  it('selects usage only with organization ownership', () => {
    expect(parseOptionalWebFeatures('organization,usage')).toEqual([
      'organization',
      'usage',
    ]);
    expect(() => parseOptionalWebFeatures('usage')).toThrow('requires "organization"');
  });

  it.each([
    ' organization',
    'organization ',
    'Organization',
    'organization,organization',
    'billing',
  ])('rejects ambiguous or unknown selection %s', selection => {
    expect(() => parseOptionalWebFeatures(selection)).toThrow();
  });

  it('requires feature dependencies explicitly', () => {
    expect(() => parseOptionalWebFeatures('permission')).toThrow('requires "organization"');
  });
});
