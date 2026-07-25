import { Activity, Cloud, Gauge, PackageCheck } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { env } from '@/config/env';
import { featureManifest } from '@/config/feature-manifest';
import { ApiHealthPanel } from '@/features/system/components/api-health-panel';
import { useApiReadiness } from '@/features/system/hooks/use-api-readiness';

export function OverviewPage() {
  const { t } = useTranslation();
  const readiness = useApiReadiness();
  const apiAvailable = readiness.data?.status === 'up' || readiness.data?.status === 'degraded';

  const metrics = [
    {
      icon: Activity,
      label: t('overview.api'),
      value: readiness.isPending
        ? t('overview.apiChecking')
        : apiAvailable
          ? t('overview.apiAvailable')
          : t('overview.apiUnavailable'),
      tone: apiAvailable ? 'text-success-strong' : 'text-foreground',
    },
    {
      icon: Cloud,
      label: t('overview.delivery'),
      value: t('overview.deliveryValue'),
      tone: 'text-brand-strong',
    },
    {
      icon: Gauge,
      label: t('overview.runtime'),
      value: t('overview.runtimeValue'),
      tone: 'text-foreground',
    },
  ];

  const releaseRows = [
    {
      label: t('overview.buildMode'),
      value: import.meta.env.PROD ? t('common.production') : t('common.development'),
    },
    {
      label: t('overview.basePath'),
      value: import.meta.env.BASE_URL,
    },
    {
      label: t('overview.apiBase'),
      value: env.API_BASE_URL,
    },
    {
      label: t('overview.featureCount'),
      value: String(Object.keys(featureManifest).length),
    },
  ];

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('overview.eyebrow')}</p>
        <h1>{t('overview.title')}</h1>
        <p>{t('overview.description')}</p>
      </header>

      <div className="metric-grid">
        {metrics.map((metric) => {
          const Icon = metric.icon;
          return (
            <article className="metric-card" key={metric.label}>
              <Icon className="size-4 text-subtle" aria-hidden="true" />
              <p>{metric.label}</p>
              <strong className={metric.tone}>{metric.value}</strong>
            </article>
          );
        })}
      </div>

      <div className="content-grid">
        <ApiHealthPanel />
        <section className="panel" aria-labelledby="release-profile-title">
          <div className="panel-header">
            <div className="flex items-start gap-3">
              <span className="icon-surface" aria-hidden="true">
                <PackageCheck className="size-4" />
              </span>
              <div>
                <h2 id="release-profile-title" className="panel-title">
                  {t('overview.releaseTitle')}
                </h2>
                <p className="panel-description">{t('overview.releaseDescription')}</p>
              </div>
            </div>
          </div>
          <dl className="divide-y divide-border">
            {releaseRows.map((row) => (
              <div className="definition-row" key={row.label}>
                <dt>{row.label}</dt>
                <dd>{row.value}</dd>
              </div>
            ))}
          </dl>
        </section>
      </div>
    </div>
  );
}
