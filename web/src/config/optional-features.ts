export const OPTIONAL_WEB_FEATURES = [
  'organization',
  'permission',
  'notification',
  'asset',
  'setting',
  'usage',
  'webhook',
] as const;

export type OptionalWebFeature = (typeof OPTIONAL_WEB_FEATURES)[number];

const knownFeatures = new Set<string>(OPTIONAL_WEB_FEATURES);
const featureDependencies: Partial<Record<OptionalWebFeature, readonly OptionalWebFeature[]>> = {
  permission: ['organization'],
  setting: ['organization'],
  usage: ['organization'],
  webhook: ['organization'],
};

export function parseOptionalWebFeatures(value: string): readonly OptionalWebFeature[] {
  if (value === '') {
    return [];
  }

  const selected = value.split(',');
  const seen = new Set<string>();

  for (const feature of selected) {
    if (feature.length === 0 || feature !== feature.trim() || !/^[a-z][a-z0-9-]*$/.test(feature)) {
      throw new Error(`Optional Web feature "${feature}" must use a canonical lowercase name`);
    }
    if (seen.has(feature)) {
      throw new Error(`Duplicate optional Web feature "${feature}"`);
    }
    if (!knownFeatures.has(feature)) {
      throw new Error(
        `Unknown optional Web feature "${feature}" (available: ${OPTIONAL_WEB_FEATURES.join(', ')})`
      );
    }
    seen.add(feature);
  }

  for (const feature of selected as OptionalWebFeature[]) {
    for (const dependency of featureDependencies[feature] ?? []) {
      if (!seen.has(dependency)) {
        throw new Error(`Optional Web feature "${feature}" requires "${dependency}"`);
      }
    }
  }

  return selected as OptionalWebFeature[];
}
