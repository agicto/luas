import type { SiteMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  nav: {
    home: 'Home',
    console: 'Console',
  },
  footer: {
    tagline: 'Full-stack scaffold for the AI era',
    copyright: '© {year} Luas. All rights reserved.',
  },
  hero: {
    eyebrow: 'Enterprise scaffold for AI-assisted teams',
    titlePrefix: 'Build modern applications faster',
    titleAccent: 'with clear architecture',
    description:
      'Luas keeps the Next.js Web and Go API independently deployable and aligned through explicit contracts, with authentication, internationalization, and engineering governance built in.',
    primaryAction: 'Get started',
    secondaryAction: 'View console',
  },
  features: {
    title: 'The foundations your business needs',
    description:
      'A replaceable, verifiable enterprise starting point with clear ownership boundaries.',
    authentication: {
      title: 'Authentication starter',
      description:
        'A complete example covering sign-in, registration, sessions, and protected routes.',
    },
    console: {
      title: 'Replaceable console',
      description: 'An authenticated workspace with settings and explicit devtool boundaries.',
    },
    i18n: {
      title: 'Type-safe internationalization',
      description:
        'Route-scoped messages with translation keys and scopes derived from the message tree.',
    },
    ui: {
      title: 'Semantic UI foundation',
      description: 'Design tokens and accessible components that support continued growth.',
    },
  },
  techStack: {
    title: 'Modern technology stack',
    description:
      'Mature tools selected for runtime performance, type safety, and developer experience.',
  },
  cta: {
    title: 'Ready to start building?',
    description:
      'Begin your next project with Luas contracts, authentication, and a replaceable console.',
    primaryAction: 'Start for free',
    githubAction: 'View on GitHub',
  },
} as const satisfies LocaleMessageShape<SiteMessages>;

export default messages;
