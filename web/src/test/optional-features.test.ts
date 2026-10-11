import { describe, expect, it } from 'vitest';

import { OPTIONAL_WEB_FEATURES, parseOptionalWebFeatures } from '@/config/optional-features';

import { CORE_OPTIONAL_WEB_FEATURES, PLUGIN_WEB_FEATURES } from './plugin-web-features';

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
    expect(parseOptionalWebFeatures('organization,setting')).toEqual(['organization', 'setting']);
    expect(() => parseOptionalWebFeatures('setting')).toThrow('requires "organization"');
  });

  it('selects usage only with organization ownership', () => {
    expect(parseOptionalWebFeatures('organization,usage')).toEqual(['organization', 'usage']);
    expect(() => parseOptionalWebFeatures('usage')).toThrow('requires "organization"');
  });

  it('selects webhook only with organization ownership', () => {
    expect(parseOptionalWebFeatures('organization,webhook')).toEqual(['organization', 'webhook']);
    expect(() => parseOptionalWebFeatures('webhook')).toThrow('requires "organization"');
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

  it('keeps the core feature list of the guard tests registered', () => {
    expect(OPTIONAL_WEB_FEATURES).toEqual(expect.arrayContaining([...CORE_OPTIONAL_WEB_FEATURES]));
  });

  it('selects every installed feature, including plugin features, with its dependencies', () => {
    expect(parseOptionalWebFeatures(OPTIONAL_WEB_FEATURES.join(','))).toEqual(
      OPTIONAL_WEB_FEATURES
    );
    for (const feature of PLUGIN_WEB_FEATURES) {
      expect(feature).toMatch(/^[a-z][a-z0-9-]*$/);
    }
  });

  it('requires feature dependencies explicitly', () => {
    expect(() => parseOptionalWebFeatures('permission')).toThrow('requires "organization"');
  });
});
