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
    kind: 'optional',
    routes: ['/login'],
  },
  users: {
    kind: 'optional',
    routes: ['/console/users'],
  },
  audit: {
    kind: 'optional',
    routes: ['/console/audit'],
  },
} as const;

export type FeatureName = keyof typeof featureManifest;
