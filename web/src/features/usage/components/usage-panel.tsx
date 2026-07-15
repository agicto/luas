'use client';

import { useMemo } from 'react';
import { useLocale } from 'next-intl';
import { AlertCircle, RefreshCw } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useOrganizationUsage, useUserUsage } from '@/features/usage/hooks/use-usage';
import type { UsageMetric, UsageSummary } from '@/features/usage/types';
import { resolveUsageErrorKey } from '@/features/usage/utils/usage-error';
import { useT } from '@/i18n';

interface UsagePanelStateProps {
  values: UsageSummary[] | undefined;
  error: unknown;
  pending: boolean;
  organization: boolean;
  refetch: () => void;
}

export function UserUsagePanel() {
  const query = useUserUsage();
  return (
    <UsagePanelState
      values={query.data}
      error={query.error}
      pending={query.isPending}
      organization={false}
      refetch={() => void query.refetch()}
    />
  );
}

export function OrganizationUsagePanel({ organizationId }: { organizationId: number }) {
  const query = useOrganizationUsage(organizationId);
  return (
    <UsagePanelState
      values={query.data}
      error={query.error}
      pending={query.isPending}
      organization
      refetch={() => void query.refetch()}
    />
  );
}

function UsagePanelState({
  values,
  error,
  pending,
  organization,
  refetch,
}: UsagePanelStateProps) {
  const t = useT('usage');
  const locale = useLocale();
  const number = useMemo(() => new Intl.NumberFormat(locale), [locale]);
  const date = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeZone: 'UTC' }),
    [locale]
  );

  if (pending) return <UsagePanelSkeleton />;
  if (error || !values) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="size-4" aria-hidden="true" />
        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
          <span>{t(resolveUsageErrorKey(error))}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            icon={<RefreshCw className="size-3.5" />}
            onClick={refetch}
          >
            {t('retry')}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <section className="py-2" aria-labelledby={organization ? 'organization-usage' : 'user-usage'}>
      <div className="mb-5 border-b pb-4">
        <h2
          id={organization ? 'organization-usage' : 'user-usage'}
          className="text-base font-semibold"
        >
          {t(organization ? 'organization.title' : 'user.title')}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {t(organization ? 'organization.description' : 'user.description')}
        </p>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('columns.metric')}</TableHead>
            <TableHead>{t('columns.used')}</TableHead>
            <TableHead>{t('columns.limit')}</TableHead>
            <TableHead>{t('columns.remaining')}</TableHead>
            <TableHead>{t('columns.period')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {values.map(item => {
            const progress =
              item.limit === null || item.limit === 0
                ? null
                : Math.min((item.used / item.limit) * 100, 100);
            return (
              <TableRow key={item.metric}>
                <TableCell className="min-w-56 whitespace-normal py-3">
                  <div className="font-medium">{metricLabel(t, item.metric)}</div>
                  <div className="mt-0.5 font-mono text-xs text-muted-foreground">
                    {item.metric}
                  </div>
                </TableCell>
                <TableCell className="min-w-36 py-3">
                  <div className="font-variant-numeric tabular-nums">
                    {number.format(item.used)} {unitLabel(t, item.unit)}
                  </div>
                  {progress === null ? null : (
                    <div
                      className="mt-2 h-1.5 w-28 overflow-hidden rounded-sm bg-muted"
                      role="progressbar"
                      aria-label={metricLabel(t, item.metric)}
                      aria-valuemin={0}
                      aria-valuemax={item.limit ?? 0}
                      aria-valuenow={Math.min(item.used, item.limit ?? item.used)}
                    >
                      <div
                        className={item.over_limit ? 'h-full bg-destructive' : 'h-full bg-primary'}
                        style={{ width: `${progress}%` }}
                      />
                    </div>
                  )}
                </TableCell>
                <TableCell className="tabular-nums">
                  {item.limit === null ? (
                    <Badge variant="outline">{t('unlimited')}</Badge>
                  ) : (
                    number.format(item.limit)
                  )}
                </TableCell>
                <TableCell className="tabular-nums">
                  {item.remaining === null ? (
                    t('notApplicable')
                  ) : item.over_limit ? (
                    <Badge variant="destructive">
                      {t('overage', { count: number.format(item.overage) })}
                    </Badge>
                  ) : (
                    number.format(item.remaining)
                  )}
                </TableCell>
                <TableCell className="min-w-44 text-muted-foreground">
                  {date.format(new Date(item.period_start))} - {date.format(new Date(item.period_end))}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </section>
  );
}

function metricLabel(t: ReturnType<typeof useT<'usage'>>, metric: UsageMetric): string {
  switch (metric) {
    case 'api.requests':
      return t('metrics.apiRequests');
    case 'ai.input_tokens':
      return t('metrics.aiInputTokens');
    case 'ai.output_tokens':
      return t('metrics.aiOutputTokens');
    case 'asset.transfer_bytes':
      return t('metrics.assetTransferBytes');
    case 'workflow.runs':
      return t('metrics.workflowRuns');
  }
}

function unitLabel(t: ReturnType<typeof useT<'usage'>>, unit: UsageSummary['unit']): string {
  switch (unit) {
    case 'request':
      return t('units.request');
    case 'token':
      return t('units.token');
    case 'byte':
      return t('units.byte');
    case 'run':
      return t('units.run');
  }
}

function UsagePanelSkeleton() {
  return (
    <div className="space-y-4 py-2" aria-hidden="true">
      <Skeleton className="h-12 w-full max-w-xl" />
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-14 w-full" />
    </div>
  );
}
