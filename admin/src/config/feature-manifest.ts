export const featureManifest = {
  system: {
    kind: 'core',
    routes: ['/console'],
  },
  preferences: {
    kind: 'core',
    routes: ['/console/preferences'],
  },
  operatorSession: {
    kind: 'core',
    routes: ['/login'],
  },
} as const;

export type FeatureName = keyof typeof featureManifest;
