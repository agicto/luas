# Dashboard Stats Block

KPI cards for dashboards and overview pages: label, value, optional icon, an optional trend, a
secondary line, and skeleton cards while loading. The trend is announced in words and colored by
meaning (`positive`), so "error rate up" can be red while "revenue up" is green.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-dashboard-stats
```

Installs `web/src/components/blocks/dashboard-stats/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy.

## Usage

```tsx
import { Activity, Users } from 'lucide-react';

import { DashboardStats } from '@/components/blocks/dashboard-stats';
import { getT } from '@/i18n/server';

export async function UsageOverview({ usage }: { usage: { members: number; requests: number } }) {
  const t = await getT('console');
  return (
    <DashboardStats
      title={t('overview.title')}
      items={[
        { id: 'members', label: t('overview.members'), value: usage.members, icon: <Users /> },
        {
          id: 'requests',
          label: t('overview.requests'),
          value: usage.requests.toLocaleString(),
          icon: <Activity />,
          trend: { value: '+12%', direction: 'up', label: t('overview.requestsTrend', { change: '12%' }) },
        },
      ]}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `items` | `StatItem[]` | `id`, `label`, `value`, optional `icon`, `trend`, `description` |
| `trend` | `StatTrend` | `value` (visible), `direction` (`up`, `down`, `flat`), `label` (announced), optional `positive` |
| `title`, `description` | `ReactNode` | Optional header; `title` names the region |
| `loading`, `loadingLabel`, `loadingCount` | | Skeleton cards, `aria-busy`, and an announced status |
| `columns` | `2 \| 3 \| 4` | Maximum columns on wide screens (default 4) |

See `INTEGRATION.md` for placement, i18n, and verification.
