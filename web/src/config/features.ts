import { env } from './env';
import {
  parseOptionalWebFeatures,
  type OptionalWebFeature,
} from './optional-features';

const activeFeatures = new Set(
  parseOptionalWebFeatures(env.NEXT_PUBLIC_OPTIONAL_FEATURES)
);

export function isWebFeatureEnabled(feature: OptionalWebFeature): boolean {
  return activeFeatures.has(feature);
}
