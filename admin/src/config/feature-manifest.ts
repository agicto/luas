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
  appSettings: {
    kind: 'optional',
    routes: ['/console/settings'],
  },
  organizations: {
    kind: 'optional',
    routes: ['/console/organizations', '/console/organizations/$organizationId'],
  },
  webhooks: {
    kind: 'optional',
    routes: ['/console/organizations/$organizationId'],
  },
  notificationDeliveries: {
    kind: 'optional',
    routes: ['/console/notifications'],
  },
} as const;

export type FeatureName = keyof typeof featureManifest;
