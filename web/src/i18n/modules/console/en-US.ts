import type { ConsoleMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  notifications: 'Notifications',
  profile: 'Profile',
  returnToSite: 'Return to site',
  greeting: {
    hello: 'Hello, {name}',
    late: 'Up late, {name}',
    morning: 'Good morning, {name}',
    afternoon: 'Good afternoon, {name}',
    evening: 'Good evening, {name}',
  },
  home: {
    defaultUser: 'there',
    welcomeDescription:
      'This is the replaceable Luas console starter. Replace it with your business workspace.',
    openSettings: 'Open settings',
    open: 'Open',
    nextSteps: 'Next steps',
    quickLinks: {
      apiDocs: {
        title: 'API docs',
        description: 'Read the OpenAPI specification for the Luas Go backend.',
      },
      styleguide: {
        title: 'Styleguide',
        description: 'Browse the design system and component gallery.',
      },
      i18nTest: {
        title: 'Internationalization test',
        description: 'Inspect the next-intl translation tree.',
      },
    },
    steps: {
      apiBefore: 'Point',
      apiAfter: 'at a contract-compatible production endpoint or adapter.',
      replaceBefore: 'Replace this page (',
      replaceAfter: ') with your real business workspace.',
      featuresBefore: 'Add new features to',
      featuresBetween: 'and expose them through routes under',
      featuresAfter: '.',
    },
  },
} as const satisfies LocaleMessageShape<ConsoleMessages>;

export default messages;
