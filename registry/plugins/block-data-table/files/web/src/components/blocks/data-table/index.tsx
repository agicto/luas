/**
 * @block DataTable
 * @description A sortable, paginated table shell, generic over the row type, with an empty state.
 * Sorting and pagination run on the rows passed in; the block fetches nothing. All copy arrives
 * through props so the caller passes translated strings. Cell renderers are functions, so render
 * the block from a Client Component.
 */
'use client';

import * as React from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/utils';

export type SortDirection = 'asc' | 'desc';

export interface DataTableSort {
  columnId: string;
  direction: SortDirection;
}

export type DataTableSortValue = string | number | boolean | Date | null | undefined;

export interface DataTableColumn<Row> {
  id: string;
  header: React.ReactNode;
  cell: (row: Row) => React.ReactNode;
  /** Makes the column sortable by this value. */
  sortValue?: (row: Row) => DataTableSortValue;
  align?: 'start' | 'end';
  /** Marks the cell as the row header, usually the first, identifying column. */
  rowHeader?: boolean;
  className?: string;
}

export interface DataTableLabels {
  /** Accessible name of the pagination navigation. */
  pagination: string;
  previousPage: string;
  nextPage: string;
  /** Visible and announced pagination status, for example "Page 2 of 5". */
  pageStatus: (page: number, pageCount: number) => string;
}

export interface DataTableProps<Row> {
  columns: DataTableColumn<Row>[];
  rows: Row[];
  getRowId: (row: Row) => string;
  /** Accessible name of the table; shown under the table unless captionHidden is set. */
  caption: string;
  captionHidden?: boolean;
  labels: DataTableLabels;
  /** Rendered in place of the body when there are no rows. */
  emptyState: React.ReactNode;
  /** Rows per page; pass 0 to disable pagination. */
  pageSize?: number;
  initialSort?: DataTableSort;
  onSortChange?: (sort: DataTableSort | null) => void;
  className?: string;
}

function compareValues(left: DataTableSortValue, right: DataTableSortValue): number {
  if (left === right) return 0;
  if (left === null || left === undefined) return 1;
  if (right === null || right === undefined) return -1;
  if (left instanceof Date && right instanceof Date) return left.getTime() - right.getTime();
  if (typeof left === 'number' && typeof right === 'number') return left - right;
  return String(left).localeCompare(String(right), undefined, { numeric: true });
}

const ariaSort = { asc: 'ascending', desc: 'descending' } as const;

export function DataTable<Row>({
  columns,
  rows,
  getRowId,
  caption,
  captionHidden = false,
  labels,
  emptyState,
  pageSize = 10,
  initialSort,
  onSortChange,
  className,
}: DataTableProps<Row>) {
  const [sort, setSort] = React.useState<DataTableSort | null>(initialSort ?? null);
  const [page, setPage] = React.useState(1);

  const sortedRows = React.useMemo(() => {
    const column = sort ? columns.find(candidate => candidate.id === sort.columnId) : undefined;
    if (!sort || !column?.sortValue) return rows;
    const value = column.sortValue;
    const factor = sort.direction === 'asc' ? 1 : -1;
    return [...rows].sort((left, right) => factor * compareValues(value(left), value(right)));
  }, [columns, rows, sort]);

  const paginated = pageSize > 0;
  const pageCount = paginated ? Math.max(1, Math.ceil(sortedRows.length / pageSize)) : 1;
  const currentPage = Math.min(page, pageCount);
  const visibleRows = paginated
    ? sortedRows.slice((currentPage - 1) * pageSize, currentPage * pageSize)
    : sortedRows;

  function toggleSort(columnId: string) {
    const next: DataTableSort | null =
      sort?.columnId !== columnId
        ? { columnId, direction: 'asc' }
        : sort.direction === 'asc'
          ? { columnId, direction: 'desc' }
          : null;
    setSort(next);
    setPage(1);
    onSortChange?.(next);
  }

  return (
    <div className={cn('flex flex-col gap-3', className)}>
      <div className="rounded-lg border bg-card">
        <Table>
          <TableCaption className={cn('mb-3', captionHidden && 'sr-only')}>{caption}</TableCaption>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {columns.map(column => {
                const direction = sort?.columnId === column.id ? sort.direction : undefined;
                const SortIcon =
                  direction === 'asc' ? ArrowUp : direction === 'desc' ? ArrowDown : ArrowUpDown;
                return (
                  <TableHead
                    key={column.id}
                    scope="col"
                    aria-sort={
                      column.sortValue ? (direction ? ariaSort[direction] : 'none') : undefined
                    }
                    className={cn('px-3', column.align === 'end' && 'text-right', column.className)}
                  >
                    {column.sortValue ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        noScale
                        className={cn(
                          '-mx-2 text-foreground',
                          column.align === 'end' && 'ml-auto flex'
                        )}
                        onClick={() => toggleSort(column.id)}
                        icon={
                          <SortIcon
                            className={cn('size-3.5', !direction && 'text-muted-foreground')}
                            aria-hidden="true"
                          />
                        }
                        iconPosition="right"
                      >
                        {column.header}
                      </Button>
                    ) : (
                      column.header
                    )}
                  </TableHead>
                );
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {visibleRows.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columns.length} className="p-0 whitespace-normal">
                  {emptyState}
                </TableCell>
              </TableRow>
            ) : (
              visibleRows.map(row => (
                <TableRow key={getRowId(row)}>
                  {columns.map(column => {
                    const cellClassName = cn(
                      'px-3',
                      column.align === 'end' && 'text-right tabular-nums',
                      column.className
                    );
                    return column.rowHeader ? (
                      <th
                        key={column.id}
                        scope="row"
                        data-slot="table-cell"
                        className={cn(
                          'p-2 text-left align-middle font-medium whitespace-nowrap',
                          cellClassName
                        )}
                      >
                        {column.cell(row)}
                      </th>
                    ) : (
                      <TableCell key={column.id} className={cellClassName}>
                        {column.cell(row)}
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {paginated && sortedRows.length > 0 ? (
        <nav
          aria-label={labels.pagination}
          className="flex flex-wrap items-center justify-between gap-2"
        >
          <p className="text-sm text-muted-foreground" aria-live="polite">
            {labels.pageStatus(currentPage, pageCount)}
          </p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={currentPage <= 1}
              onClick={() => setPage(currentPage - 1)}
              icon={<ChevronLeft className="size-4" aria-hidden="true" />}
            >
              {labels.previousPage}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={currentPage >= pageCount}
              onClick={() => setPage(currentPage + 1)}
              icon={<ChevronRight className="size-4" aria-hidden="true" />}
              iconPosition="right"
            >
              {labels.nextPage}
            </Button>
          </div>
        </nav>
      ) : null}
    </div>
  );
}
