'use client';

import { useMemo, useState } from 'react';
import {
  Clock3,
  Database,
  ExternalLink,
  ListChecks,
  RefreshCw,
  Sparkles,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useT } from '@/i18n';
import { cn } from '@/utils';
import {
  useCreateTrendSyncRun,
  useTrends,
  useTrendSummary,
} from '@/features/trends/hooks/use-trends';
import type { TrendItem, TrendStatus } from '@/features/trends/types';

type TrendTab = 'all' | TrendStatus;

interface StatItem {
  label: string;
  value: number;
  icon: typeof Database;
  tone: 'info' | 'success' | 'warning' | 'highlight';
}

const statusOrder: TrendTab[] = ['all', 'candidate', 'new', 'selected', 'rejected'];

/**
 * @component Trend Console
 * @category Feature
 * @status Beta
 * @description Operational console for synced, scored trend items.
 * @usage Render inside the protected console route.
 * @example
 * <TrendConsole />
 */
export function TrendConsole() {
  const t = useT('trends');
  const [status, setStatus] = useState<TrendTab>('candidate');
  const [search, setSearch] = useState('');
  const statusFilter = status === 'all' ? undefined : status;

  const query = useMemo(
    () => ({
      status: statusFilter,
      search: search.trim() || undefined,
      page: 1,
      per_page: 50,
    }),
    [search, statusFilter]
  );

  const trendsQuery = useTrends(query);
  const summaryQuery = useTrendSummary();
  const syncRun = useCreateTrendSyncRun();

  const stats = summaryQuery.data?.stats;
  const source = summaryQuery.data?.source;
  const trends = trendsQuery.data?.items ?? [];

  const statItems: StatItem[] = [
    {
      label: t('stats.total'),
      value: stats?.total ?? 0,
      icon: Database,
      tone: 'info',
    },
    {
      label: t('stats.candidate'),
      value: stats?.candidate ?? 0,
      icon: Sparkles,
      tone: 'success',
    },
    {
      label: t('stats.new'),
      value: stats?.new ?? 0,
      icon: Clock3,
      tone: 'highlight',
    },
    {
      label: t('stats.queued'),
      value: stats?.queued_jobs ?? 0,
      icon: ListChecks,
      tone: 'warning',
    },
  ];

  const isLoading = trendsQuery.isLoading || summaryQuery.isLoading;

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-6 md:px-8">
      <header className="flex flex-col gap-4 border-b border-border-subtle pb-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal text-text-main">{t('title')}</h1>
          <p className="mt-1 text-sm text-text-muted">{t('description')}</p>
        </div>

        <div className="flex flex-col items-start gap-2 text-sm text-text-muted sm:flex-row sm:items-center">
          <div className="flex min-h-9 items-center gap-2 rounded-md border border-border-subtle bg-bg-surface px-3">
            <Database className="size-4 text-info" />
            <span className="font-medium text-text-main">
              {source?.name ?? t('noSource')}
            </span>
            {source?.poll_interval_minutes ? (
              <span className="text-xs">
                {t('interval', { count: source.poll_interval_minutes })}
              </span>
            ) : null}
          </div>
          <Button
            size="sm"
            icon={<RefreshCw className={cn('size-4', syncRun.isPending ? 'animate-spin' : '')} />}
            loading={syncRun.isPending}
            onClick={() => syncRun.mutate()}
          >
            {syncRun.isPending ? t('refreshing') : t('syncNow')}
          </Button>
        </div>
      </header>

      <section className="grid gap-3 md:grid-cols-4">
        {statItems.map((item) => (
          <StatCard key={item.label} item={item} loading={summaryQuery.isLoading} />
        ))}
      </section>

      <section className="flex flex-col gap-3 rounded-lg border border-border-subtle bg-bg-surface p-3">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <Tabs value={status} onValueChange={(value) => setStatus(value as TrendTab)}>
            <TabsList className="h-auto flex-wrap justify-start">
              {statusOrder.map((value) => (
                <TabsTrigger key={value} value={value} className="min-h-8">
                  {t(`tabs.${value}`)}
                  <span className="ml-1 rounded-sm bg-bg-subtle px-1.5 py-0.5 text-[11px] text-text-muted">
                    {countForTab(value, stats)}
                  </span>
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <div className="w-full lg:w-[360px]">
            <SearchInput
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t('searchPlaceholder')}
            />
          </div>
        </div>

        <div className="overflow-hidden rounded-md border border-border-subtle">
          <Table>
            <TableHeader>
              <TableRow className="bg-bg-subtle/70">
                <TableHead className="w-[42%] px-3">{t('table.topic')}</TableHead>
                <TableHead>{t('table.channel')}</TableHead>
                <TableHead className="min-w-[190px]">{t('table.score')}</TableHead>
                <TableHead>{t('table.audience')}</TableHead>
                <TableHead>{t('table.status')}</TableHead>
                <TableHead>{t('table.time')}</TableHead>
                <TableHead className="text-right">{t('table.action')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? <TrendSkeletonRows /> : null}
              {!isLoading && trends.length === 0 ? <TrendEmptyRow /> : null}
              {!isLoading
                ? trends.map((trend) => (
                    <TrendRow key={trend.id} trend={trend} />
                  ))
                : null}
            </TableBody>
          </Table>
        </div>
      </section>
    </div>
  );
}

function StatCard({ item, loading }: { item: StatItem; loading: boolean }) {
  const Icon = item.icon;
  const toneClass = {
    info: 'bg-info/10 text-info border-info/20',
    success: 'bg-success/10 text-success border-success/20',
    warning: 'bg-warning/10 text-warning border-warning/20',
    highlight: 'bg-highlight/10 text-highlight border-highlight/20',
  }[item.tone];

  return (
    <div className="rounded-lg border border-border-subtle bg-bg-surface p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-xs font-medium uppercase text-text-muted">{item.label}</p>
          {loading ? (
            <Skeleton className="mt-2 h-7 w-16" />
          ) : (
            <p className="mt-1 text-2xl font-semibold text-text-main">{item.value}</p>
          )}
        </div>
        <div className={cn('flex size-9 items-center justify-center rounded-md border', toneClass)}>
          <Icon className="size-4" />
        </div>
      </div>
    </div>
  );
}

function TrendRow({ trend }: { trend: TrendItem }) {
  const t = useT('trends');
  const totalScore = trend.scores.total_score ?? 0;

  return (
    <TableRow>
      <TableCell className="px-3 whitespace-normal">
        <div className="max-w-[620px]">
          <div className="flex flex-wrap items-center gap-2">
            <p className="line-clamp-2 text-sm font-medium leading-5 text-text-main">
              {trend.title}
            </p>
            {trend.significance ? (
              <Badge variant="outline" className="border-warning/30 bg-warning/10 text-warning">
                {trend.significance}
              </Badge>
            ) : null}
          </div>
          {trend.summary ? (
            <p className="mt-1 line-clamp-2 text-xs leading-5 text-text-muted">
              {trend.summary}
            </p>
          ) : null}
          {trend.recommended_angle ? (
            <p className="mt-2 line-clamp-2 text-xs leading-5 text-text-subtle">
              {trend.recommended_angle}
            </p>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <span className="text-sm text-text-subtle">{trend.channel || '-'}</span>
      </TableCell>
      <TableCell>
        <div className="flex min-w-[170px] flex-col gap-1.5">
          <div className="flex items-center justify-between text-xs">
            <span className="font-medium text-text-main">{t('score.total')}</span>
            <span className="font-mono text-text-main">{totalScore}</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-bg-subtle">
            <div
              className="h-full rounded-full bg-success"
              style={{ width: `${Math.min(Math.max(totalScore, 0), 20) * 5}%` }}
            />
          </div>
          <div className="grid grid-cols-5 gap-1 text-[11px] text-text-muted">
            <span>{t('score.h')} {trend.scores.h_score ?? '-'}</span>
            <span>{t('score.k')} {trend.scores.k_score ?? '-'}</span>
            <span>{t('score.r')} {trend.scores.r_score ?? '-'}</span>
            <span>{t('score.brand')} {trend.scores.brand_fit_score ?? '-'}</span>
            <span>{t('score.risk')} {trend.scores.risk_score ?? '-'}</span>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <span className="text-sm text-text-subtle">{trend.target_audience || '-'}</span>
      </TableCell>
      <TableCell>
        <StatusBadge status={trend.status} />
      </TableCell>
      <TableCell>
        <span className="text-xs text-text-muted">
          {formatDateTime(trend.highlighted_at ?? trend.created_at)}
        </span>
      </TableCell>
      <TableCell className="text-right">
        <Button asChild variant="ghost" size="sm" isIcon>
          <a href={trend.canonical_url} target="_blank" rel="noreferrer">
            <ExternalLink className="size-4" />
            <span className="sr-only">{t('table.action')}</span>
          </a>
        </Button>
      </TableCell>
    </TableRow>
  );
}

function StatusBadge({ status }: { status: TrendStatus }) {
  const t = useT('trends');
  const className = {
    candidate: 'border-success/30 bg-success/10 text-success',
    new: 'border-info/30 bg-info/10 text-info',
    selected: 'border-highlight/30 bg-highlight/10 text-highlight',
    rejected: 'border-destructive/30 bg-destructive/10 text-destructive',
    archived: 'border-border-subtle bg-bg-subtle text-text-muted',
  }[status];

  return (
    <Badge variant="outline" className={className}>
      {t(`status.${status}`)}
    </Badge>
  );
}

function TrendSkeletonRows() {
  return (
    <>
      {[0, 1, 2, 3, 4].map((index) => (
        <TableRow key={index}>
          <TableCell className="px-3">
            <Skeleton className="h-4 w-4/5" />
            <Skeleton className="mt-2 h-3 w-3/5" />
          </TableCell>
          <TableCell><Skeleton className="h-4 w-16" /></TableCell>
          <TableCell><Skeleton className="h-8 w-40" /></TableCell>
          <TableCell><Skeleton className="h-4 w-20" /></TableCell>
          <TableCell><Skeleton className="h-6 w-16" /></TableCell>
          <TableCell><Skeleton className="h-4 w-24" /></TableCell>
          <TableCell><Skeleton className="ml-auto h-8 w-8" /></TableCell>
        </TableRow>
      ))}
    </>
  );
}

function TrendEmptyRow() {
  const t = useT('trends');

  return (
    <TableRow>
      <TableCell colSpan={7} className="h-40 text-center">
        <div className="mx-auto flex max-w-sm flex-col items-center gap-2">
          <Sparkles className="size-8 text-text-muted" />
          <p className="text-sm font-medium text-text-main">{t('empty.title')}</p>
          <p className="text-sm text-text-muted">{t('empty.description')}</p>
        </div>
      </TableCell>
    </TableRow>
  );
}

function countForTab(tab: TrendTab, stats?: { total: number; candidate: number; new: number; selected: number; rejected: number }) {
  if (!stats) {
    return 0;
  }

  switch (tab) {
    case 'all':
      return stats.total;
    case 'candidate':
      return stats.candidate;
    case 'new':
      return stats.new;
    case 'selected':
      return stats.selected;
    case 'rejected':
      return stats.rejected;
    default:
      return 0;
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return '-';
  }

  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}
