import { useQuery } from '@tanstack/react-query';
import { Server } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import { systemService } from '@/features/system/services/system-service';

export function OperatorSystemPanel() {
  const { t } = useTranslation();
  const system = useQuery({
    queryKey: ['operator', 'system'],
    queryFn: ({ signal }) => systemService.operatorSystem(signal),
    staleTime: 15_000,
  });

  return (
    <section className="panel" aria-labelledby="operator-system-title">
      <div className="panel-header">
        <div className="flex items-start gap-3">
          <span className="icon-surface" aria-hidden="true">
            <Server className="size-4" />
          </span>
          <div>
            <h2 id="operator-system-title" className="panel-title">
              {t('system.title')}
            </h2>
            <p className="panel-description">{t('system.description')}</p>
          </div>
        </div>
      </div>
      <div className="panel-body">
        {system.isPending ? (
          <Skeleton className="h-20 w-full" />
        ) : system.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {t('system.unavailable')}
          </p>
        ) : (
          <dl className="flex flex-col gap-2 text-sm">
            <div className="definition-row">
              <dt>{t('system.database')}</dt>
              <dd>
                <StatusBadge>
                  {system.data.database === 'ok'
                    ? t('system.databaseOk')
                    : t('system.databaseDown')}
                </StatusBadge>
              </dd>
            </div>
            <div className="definition-row">
              <dt>{t('system.build')}</dt>
              <dd>
                <code>
                  {system.data.version}
                  {system.data.revision ? ` · ${system.data.revision}` : ''} ·{' '}
                  {system.data.go_version}
                </code>
              </dd>
            </div>
            <div className="definition-row">
              <dt>{t('system.starters')}</dt>
              <dd className="flex flex-wrap justify-end gap-1">
                {system.data.starters.map((name) => (
                  <StatusBadge key={name}>{name}</StatusBadge>
                ))}
              </dd>
            </div>
          </dl>
        )}
      </div>
    </section>
  );
}
