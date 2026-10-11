# Onboarding Steps Block

A getting-started checklist: a progress bar with an accessible label and an ordered list of steps
that are complete, current, or upcoming. Each status is announced in words, the current step gets
`aria-current="step"`, and only the current step shows its action.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-onboarding-steps
```

Installs `web/src/components/blocks/onboarding-steps/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy. Status comes from you.

## Usage

```tsx
import Link from 'next/link';

import { OnboardingSteps, type OnboardingStep } from '@/components/blocks/onboarding-steps';
import { Button } from '@/components/ui/button';
import { getT } from '@/i18n/server';

export async function GettingStarted({ hasProfile, memberCount }: { hasProfile: boolean; memberCount: number }) {
  const t = await getT('console');
  const steps: OnboardingStep[] = [
    { id: 'profile', title: t('onboarding.profile'), status: hasProfile ? 'complete' : 'current' },
    {
      id: 'invite',
      title: t('onboarding.invite'),
      description: t('onboarding.inviteDescription'),
      status: !hasProfile ? 'upcoming' : memberCount > 1 ? 'complete' : 'current',
      action: <Button asChild size="sm"><Link href="/console/organizations">{t('onboarding.inviteAction')}</Link></Button>,
    },
  ];
  const done = steps.filter((step) => step.status === 'complete').length;
  return (
    <OnboardingSteps
      title={t('onboarding.title')}
      steps={steps}
      labels={{
        progress: t('onboarding.progress', { done, total: steps.length }),
        complete: t('onboarding.statusComplete'),
        current: t('onboarding.statusCurrent'),
        upcoming: t('onboarding.statusUpcoming'),
      }}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `title`, `description` | `ReactNode` | `title` names the region |
| `steps` | `OnboardingStep[]` | `id`, `title`, `description`, `status`, `action` (shown on the current step) |
| `labels` | `OnboardingStepsLabels` | Formatted `progress`, and the announced `complete`, `current`, `upcoming` |
| `headingLevel` | `'h2' \| 'h3'` | Step titles use the next level |

See `INTEGRATION.md` for placement, i18n, and verification.
