import { Activity, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { ApiError } from '@/http/client';
import { useApiReadiness } from '@/features/system/hooks/use-api-readiness';

export function ApiHealthPanel() {
  const { t, i18n } = useTranslation();
  const query = useApiReadiness();
  const available = query.data?.status === 'up' || query.data?.status === 'degraded';
  const error = query.error instanceof ApiError ? query.error : undefined;
  const readinessStatus =
    query.data?.status === 'up' ? t('overview.statusUp') : t('overview.statusDegraded');

  const label = query.isPending
    ? t('overview.apiChecking')
    : available
      ? t('overview.apiAvailable')
      : t('overview.apiUnavailable');

  return (
    <section className="panel" aria-labelledby="api-readiness-title">
      <div className="panel-header">
        <div className="flex min-w-0 items-start gap-3">
          <span className="icon-surface" aria-hidden="true">
            <Activity className="size-4" />
          </span>
          <div className="min-w-0">
            <h2 id="api-readiness-title" className="panel-title">
              {t('overview.readinessTitle')}
            </h2>
            <p className="panel-description">{t('overview.readinessDescription')}</p>
          </div>
        </div>
        <Button
          aria-label={t('overview.refresh')}
          title={t('overview.refresh')}
          variant="outline"
          size="icon"
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <RefreshCw className={query.isFetching ? 'size-4 animate-spin' : 'size-4'} />
        </Button>
      </div>

      <div className="panel-body">
        <div className="flex flex-wrap items-center gap-3">
          <StatusBadge
            tone={query.isPending ? 'neutral' : available ? 'success' : 'danger'}
            aria-live="polite"
          >
            {label}
          </StatusBadge>
          {query.dataUpdatedAt > 0 ? (
            <span className="text-xs text-subtle">
              {t('overview.checked', {
                time: new Intl.DateTimeFormat(i18n.language, {
                  hour: '2-digit',
                  minute: '2-digit',
                  second: '2-digit',
                }).format(query.dataUpdatedAt),
              })}
            </span>
          ) : null}
        </div>

        <p className="mt-4 text-sm text-subtle">
          {query.isPending
            ? t('overview.waiting')
            : available
              ? t('overview.statusValue', { status: readinessStatus })
              : t('overview.unreachable')}
        </p>

        {error?.requestId ? (
          <p className="mt-2 font-mono text-xs text-subtle">
            {t('overview.requestId')}: {error.requestId}
          </p>
        ) : null}
      </div>
    </section>
  );
}
