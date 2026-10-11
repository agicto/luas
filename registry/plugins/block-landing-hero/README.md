# Landing Hero Block

The top of a public landing page: an eyebrow badge, the page heading (`h1`), supporting copy,
primary and secondary calls to action, optional media beside the copy on wide screens, and a
responsive feature grid with decorative icons.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-landing-hero
```

Installs `web/src/components/blocks/landing-hero/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy.

## Usage

```tsx
import Link from 'next/link';
import { Globe, Shield, Terminal } from 'lucide-react';

import { LandingHero } from '@/components/blocks/landing-hero';
import { Button } from '@/components/ui/button';
import { getT } from '@/i18n/server';

export default async function HomePage() {
  const t = await getT('site');
  return (
    <LandingHero
      eyebrow={t('hero.eyebrow')}
      title={t('hero.title')}
      description={t('hero.description')}
      primaryAction={<Button asChild size="lg"><Link href="/register">{t('hero.primary')}</Link></Button>}
      secondaryAction={<Button asChild size="lg" variant="outline"><Link href="/docs">{t('hero.secondary')}</Link></Button>}
      featuresHeading={t('features.heading')}
      featuresHeadingHidden
      features={[
        { id: 'auth', title: t('features.authentication.title'), description: t('features.authentication.description'), icon: <Shield /> },
        { id: 'console', title: t('features.console.title'), description: t('features.console.description'), icon: <Terminal /> },
        { id: 'i18n', title: t('features.i18n.title'), description: t('features.i18n.description'), icon: <Globe /> },
      ]}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `title`, `description`, `eyebrow` | `ReactNode` | `title` is the page `h1` |
| `primaryAction`, `secondaryAction` | `ReactNode` | Full width on phones, inline from `sm` |
| `media` | `ReactNode` | Image or illustration; switches to a two-column layout on large screens |
| `features` | `LandingFeature[]` | `id`, `title`, `description`, decorative `icon` |
| `featuresHeading`, `featuresHeadingHidden` | | Names the feature section; can be visually hidden |

See `INTEGRATION.md` for placement, i18n, and verification.
