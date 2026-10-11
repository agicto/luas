# Pricing Table Block

Plan cards with a name, description, preformatted price and period, a feature list that announces
"included" or "not included" in words, a call-to-action slot, and one optionally highlighted plan
with a badge. One to four plans lay out responsively; more wrap.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-pricing-table
```

Installs `web/src/components/blocks/pricing-table/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy.

## Usage

```tsx
import Link from 'next/link';

import { PricingTable } from '@/components/blocks/pricing-table';
import { Button } from '@/components/ui/button';
import { getT } from '@/i18n/server';

export async function Pricing() {
  const t = await getT('site');
  return (
    <PricingTable
      title={t('pricing.title')}
      labels={{ included: t('pricing.included'), notIncluded: t('pricing.notIncluded') }}
      plans={[
        {
          id: 'free',
          name: t('pricing.free.name'),
          price: t('pricing.free.price'),
          period: t('pricing.perMonth'),
          features: [{ id: 'projects', label: t('pricing.free.projects'), included: true }],
          action: <Button asChild variant="outline"><Link href="/register">{t('pricing.free.cta')}</Link></Button>,
        },
        {
          id: 'team',
          name: t('pricing.team.name'),
          price: t('pricing.team.price'),
          period: t('pricing.perSeat'),
          highlighted: true,
          badge: t('pricing.popular'),
          features: [{ id: 'sso', label: t('pricing.team.sso'), included: true }],
          action: <Button asChild><Link href="/register?plan=team">{t('pricing.team.cta')}</Link></Button>,
        },
      ]}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `plans` | `PricingPlan[]` | `id`, `name`, `description`, `price`, `period`, `features`, `action`, `highlighted`, `badge` |
| `labels` | `PricingTableLabels` | `included` and `notIncluded`, announced before each feature |
| `title`, `description` | `ReactNode` | Optional header; `title` names the region |
| `controls` | `ReactNode` | Between the header and the plans, for example a monthly/yearly toggle |

See `INTEGRATION.md` for placement, i18n, and verification.
