# Data Table Block

A sortable, paginated table shell, generic over the row type. Headers become sort buttons with
`aria-sort`; pagination has labelled previous/next buttons and an announced page status; an empty
state spans all columns. Sorting and pagination run on the rows you pass in.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-data-table
```

Installs `web/src/components/blocks/data-table/` (component and test). It is a Client Component
because columns hold render functions; render it from a Client Component.

## Usage

```tsx
'use client';

import { DataTable, type DataTableColumn } from '@/components/blocks/data-table';
import { EmptyState } from '@/components/blocks/empty-state'; // optional, from block-empty-state
import { useT } from '@/i18n';

interface Member { id: string; name: string; role: string; joinedAt: Date }

export function MemberTable({ members }: { members: Member[] }) {
  const t = useT('organization');
  const columns: DataTableColumn<Member>[] = [
    { id: 'name', header: t('members.name'), cell: (m) => m.name, sortValue: (m) => m.name, rowHeader: true },
    { id: 'role', header: t('members.role'), cell: (m) => m.role },
    { id: 'joined', header: t('members.joined'), cell: (m) => m.joinedAt.toLocaleDateString(), sortValue: (m) => m.joinedAt, align: 'end' },
  ];
  return (
    <DataTable
      caption={t('members.caption')}
      captionHidden
      columns={columns}
      rows={members}
      getRowId={(m) => m.id}
      pageSize={20}
      labels={{
        pagination: t('members.pagination'),
        previousPage: t('members.previous'),
        nextPage: t('members.next'),
        pageStatus: (page, pageCount) => t('members.pageStatus', { page, pageCount }),
      }}
      emptyState={<p className="p-6 text-center text-sm text-muted-foreground">{t('members.empty')}</p>}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `columns` | `DataTableColumn<Row>[]` | `id`, `header`, `cell(row)`, optional `sortValue(row)`, `align`, `rowHeader`, `className` |
| `rows`, `getRowId` | | Rows for the current data set and a stable key |
| `caption`, `captionHidden` | `string`, `boolean` | Accessible table name; hide it visually when a heading already names the table |
| `labels` | `DataTableLabels` | `pagination`, `previousPage`, `nextPage`, `pageStatus(page, pageCount)` |
| `emptyState` | `ReactNode` | Shown when `rows` is empty |
| `pageSize` | `number` | Default 10; `0` disables pagination |
| `initialSort`, `onSortChange` | | Initial order and a callback when the user changes it |

See `INTEGRATION.md` for placement, i18n, server-side data, and verification.
