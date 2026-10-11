# Empty State Block

A centered panel for the moments a page has nothing to show. Four variants share one layout, each
with its own decorative icon and tone:

| Variant | Use for | Semantics |
|---|---|---|
| `empty` (default) | A collection with no items yet | Labelled region |
| `no-results` | A search or filter that matched nothing | Labelled region |
| `error` | A failed load, with an optional request id in `detail` | `role="alert"` |
| `no-access` | Missing permission (403) | Labelled region |

```bash
cd api && go run ./cmd/fastbuild plugin:add block-empty-state
```

Installs `web/src/components/blocks/empty-state/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy.

## Usage

```tsx
import { EmptyState } from '@/components/blocks/empty-state';
import { Button } from '@/components/ui/button';
import { useT } from '@/i18n';

export function ProjectsEmpty({ onCreate }: { onCreate: () => void }) {
  const t = useT('projects');
  return (
    <EmptyState
      title={t('empty.title')}
      description={t('empty.description')}
      action={<Button onClick={onCreate}>{t('empty.create')}</Button>}
    />
  );
}

// Failed load inside an existing card:
<EmptyState
  variant="error"
  bare
  headingLevel="h3"
  title={t('loadError.title')}
  description={t('loadError.description')}
  detail={error.requestId ? t('loadError.requestId', { id: error.requestId }) : undefined}
  action={<Button variant="outline" onClick={() => refetch()}>{t('loadError.retry')}</Button>}
/>
```

| Prop | Type | Notes |
|---|---|---|
| `title`, `description` | `ReactNode` | `title` names the region |
| `variant` | `'empty' \| 'no-results' \| 'error' \| 'no-access'` | Icon, tone, and `role="alert"` for errors |
| `action`, `secondaryAction` | `ReactNode` | Stacked on phones, side by side from `sm` |
| `detail` | `ReactNode` | Small monospace detail such as a request id |
| `icon` | `ReactNode` | Replaces the variant icon (decorative) |
| `headingLevel` | `'h2' \| 'h3' \| 'h4'` | Match the surrounding outline |
| `bare` | `boolean` | Removes the dashed frame inside cards and table cells |

See `INTEGRATION.md` for placement, i18n, and verification.
