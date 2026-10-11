/**
 * @block DashboardStats
 * @description KPI cards with an optional trend per metric. All copy arrives through props so the
 * caller passes translated strings; the block fetches nothing and is safe in Server Components.
 */
import * as React from 'react';
import { Minus, TrendingDown, TrendingUp } from 'lucide-react';

import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/utils';

export type StatTrendDirection = 'up' | 'down' | 'flat';

export interface StatTrend {
  /** Short visible change, for example a formatted "+12%". */
  value: React.ReactNode;
  direction: StatTrendDirection;
  /** Full sentence for assistive technology, for example "Up 12% from last month". */
  label: string;
  /**
   * Whether the direction is good news. Revenue going up is positive; error rate going up is not.
   * Defaults to true for up, false for down, and neutral for flat.
   */
  positive?: boolean;
}

export interface StatItem {
  id: string;
  label: React.ReactNode;
  value: React.ReactNode;
  icon?: React.ReactNode;
  trend?: StatTrend;
  /** Secondary line under the value, for example the comparison period. */
  description?: React.ReactNode;
}

export interface DashboardStatsProps {
  items: StatItem[];
  title?: React.ReactNode;
  description?: React.ReactNode;
  /** Shows skeleton cards instead of items; loadingLabel is announced to screen readers. */
  loading?: boolean;
  loadingLabel?: string;
  /** Number of skeleton cards while loading. */
  loadingCount?: number;
  /** Maximum columns on wide screens. */
  columns?: 2 | 3 | 4;
  className?: string;
}

const columnClasses: Record<NonNullable<DashboardStatsProps['columns']>, string> = {
  2: 'sm:grid-cols-2',
  3: 'sm:grid-cols-2 lg:grid-cols-3',
  4: 'sm:grid-cols-2 xl:grid-cols-4',
};

function TrendBadge({ trend }: { trend: StatTrend }) {
  const Icon =
    trend.direction === 'up' ? TrendingUp : trend.direction === 'down' ? TrendingDown : Minus;
  const positive = trend.positive ?? (trend.direction === 'flat' ? undefined : trend.direction === 'up');
  const tone =
    positive === undefined ? 'text-muted-foreground' : positive ? 'text-success' : 'text-error';

  return (
    <p className={cn('flex items-center gap-1 text-sm font-medium', tone)}>
      <Icon className="size-4 shrink-0" aria-hidden="true" />
      <span aria-hidden="true">{trend.value}</span>
      <span className="sr-only">{trend.label}</span>
    </p>
  );
}

export function DashboardStats({
  items,
  title,
  description,
  loading = false,
  loadingLabel,
  loadingCount = 4,
  columns = 4,
  className,
}: DashboardStatsProps) {
  const headingId = React.useId();

  return (
    <section
      aria-labelledby={title ? headingId : undefined}
      aria-busy={loading || undefined}
      className={cn('flex flex-col gap-4', className)}
    >
      {title || description ? (
        <header className="flex flex-col gap-1">
          {title ? (
            <h2 id={headingId} className="text-lg font-semibold text-foreground">
              {title}
            </h2>
          ) : null}
          {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
        </header>
      ) : null}

      {loading ? (
        <div className={cn('grid grid-cols-1 gap-4', columnClasses[columns])}>
          {loadingLabel ? (
            <p role="status" className="sr-only">
              {loadingLabel}
            </p>
          ) : null}
          {Array.from({ length: loadingCount }, (_, index) => (
            <Card key={index} aria-hidden="true" className="gap-3 py-5">
              <CardHeader className="px-5">
                <Skeleton className="h-4 w-24" />
              </CardHeader>
              <CardContent className="flex flex-col gap-2 px-5">
                <Skeleton className="h-8 w-32" />
                <Skeleton className="h-4 w-20" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <ul role="list" className={cn('grid grid-cols-1 gap-4', columnClasses[columns])}>
          {items.map((item) => (
            <li key={item.id} className="min-w-0">
              <Card className="h-full gap-3 py-5">
                <CardHeader className="flex items-center justify-between gap-2 px-5">
                  <p className="truncate text-sm font-medium text-muted-foreground">{item.label}</p>
                  {item.icon ? (
                    <span
                      className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground [&>svg]:size-4"
                      aria-hidden="true"
                    >
                      {item.icon}
                    </span>
                  ) : null}
                </CardHeader>
                <CardContent className="flex flex-col gap-1 px-5">
                  <p className="truncate text-2xl font-semibold tracking-tight text-foreground tabular-nums">
                    {item.value}
                  </p>
                  {item.trend ? <TrendBadge trend={item.trend} /> : null}
                  {item.description ? (
                    <p className="text-xs text-muted-foreground">{item.description}</p>
                  ) : null}
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
