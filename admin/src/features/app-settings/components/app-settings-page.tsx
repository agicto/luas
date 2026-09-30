import { RotateCcw, Save, SlidersHorizontal } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  useAppSettings,
  useChangeAppSetting,
} from '@/features/app-settings/hooks/use-app-settings';
import { isInactiveStarter, settingErrorKey } from '@/features/app-settings/setting-errors';
import type { AppSetting, AppSettingValue } from '@/features/app-settings/types';
import { cn } from '@/lib/cn';

function SettingEditor({ setting }: { setting: AppSetting }) {
  const { t } = useTranslation();
  const change = useChangeAppSetting();
  const [draft, setDraft] = useState<AppSettingValue>(setting.value);
  const dirty = draft !== setting.value;

  const save = (value: AppSettingValue) =>
    change.mutate({ kind: 'set', key: setting.key, value, version: setting.version });

  let control;
  if (setting.kind === 'boolean') {
    control = (
      <div className="segmented-control" role="group" aria-label={setting.key}>
        {[true, false].map((value) => (
          <Button
            key={String(value)}
            type="button"
            size="sm"
            variant="ghost"
            aria-pressed={draft === value}
            className={cn('flex-1', draft === value && 'bg-background shadow-sm')}
            onClick={() => setDraft(value)}
          >
            {value ? t('settings.on') : t('settings.off')}
          </Button>
        ))}
      </div>
    );
  } else if (setting.kind === 'enum') {
    control = (
      <select
        aria-label={setting.key}
        className="select-input"
        value={String(draft)}
        onChange={(event) => setDraft(event.target.value)}
      >
        {(setting.options ?? []).map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  } else {
    control = (
      <Input
        aria-label={setting.key}
        type={setting.kind === 'integer' ? 'number' : 'text'}
        value={String(draft)}
        onChange={(event) =>
          setDraft(setting.kind === 'integer' ? Number(event.target.value) : event.target.value)
        }
      />
    );
  }

  return (
    <li className="flex flex-col gap-2 border-b px-4 py-3 last:border-0 md:flex-row md:items-center">
      <div className="min-w-0 md:w-1/3">
        <code className="text-sm font-medium">{setting.key}</code>
        <div className="mt-1 flex flex-wrap gap-1">
          <StatusBadge>{t(`settings.source.${setting.source}`)}</StatusBadge>
          <StatusBadge>{t('settings.version', { version: setting.version })}</StatusBadge>
        </div>
      </div>
      <div className="flex-1">{control}</div>
      <div className="flex gap-2">
        <Button
          type="button"
          size="sm"
          disabled={!dirty || change.isPending}
          onClick={() => save(draft)}
        >
          <Save aria-hidden="true" />
          {t('settings.save')}
        </Button>
        {setting.source === 'override' ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={change.isPending}
            onClick={() =>
              change.mutate({ kind: 'reset', key: setting.key, version: setting.version })
            }
          >
            <RotateCcw aria-hidden="true" />
            {t('settings.reset')}
          </Button>
        ) : null}
      </div>
      {change.isError ? (
        <p role="alert" className="text-sm text-destructive md:basis-full">
          {t(settingErrorKey(change.error))}
        </p>
      ) : null}
    </li>
  );
}

export function AppSettingsPage() {
  const { t } = useTranslation();
  const settings = useAppSettings();

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('settings.eyebrow')}</p>
        <h1>{t('settings.title')}</h1>
        <p>{t('settings.description')}</p>
      </header>
      <section className="panel" aria-labelledby="settings-title">
        <div className="panel-header">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <SlidersHorizontal className="size-4" />
            </span>
            <div>
              <h2 id="settings-title" className="panel-title">
                {t('settings.listTitle')}
              </h2>
              <p className="panel-description">{t('settings.concurrencyNote')}</p>
            </div>
          </div>
        </div>
        <div className="panel-body p-0">
          {settings.isPending ? (
            <div className="flex flex-col gap-2 p-4" aria-busy="true">
              {Array.from({ length: 3 }, (_, index) => (
                <Skeleton key={index} className="h-10 w-full" />
              ))}
            </div>
          ) : settings.isError ? (
            <p role="alert" className="p-4 text-sm text-muted-foreground">
              {isInactiveStarter(settings.error)
                ? t('settings.inactive')
                : t('settings.errors.unavailable')}
            </p>
          ) : settings.data.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">{t('settings.empty')}</p>
          ) : (
            <ul>
              {settings.data.map((setting) => (
                <SettingEditor key={`${setting.key}:${setting.version}`} setting={setting} />
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  );
}
