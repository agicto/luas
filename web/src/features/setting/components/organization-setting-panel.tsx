'use client';

import { useState } from 'react';
import { AlertCircle, Globe2, RefreshCw, RotateCcw, Save } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  useOrganizationSettings,
  useResetOrganizationSetting,
  useSetOrganizationSetting,
} from '@/features/setting/hooks/use-settings';
import { localeSettingMutationSchema } from '@/features/setting/schemas';
import type { OrganizationSetting } from '@/features/setting/types';
import { resolveSettingErrorKey } from '@/features/setting/utils/setting-error';
import { useT } from '@/i18n';

export function OrganizationSettingPanel({
  organizationId,
  canManage,
}: {
  organizationId: number;
  canManage: boolean;
}) {
  const t = useT('setting');
  const query = useOrganizationSettings(organizationId);

  if (query.isPending) return <OrganizationSettingSkeleton />;
  if (query.error || !query.data) {
    return (
      <Alert variant="destructive" className="my-6">
        <AlertCircle className="size-4" aria-hidden="true" />
        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
          <span>{t(resolveSettingErrorKey(query.error))}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            icon={<RefreshCw className="size-3.5" />}
            onClick={() => query.refetch()}
          >
            {t('actions.retry')}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }
  const locale = query.data[0];
  if (!locale) return null;
  return (
    <OrganizationSettingForm
      key={locale.version}
      organizationId={organizationId}
      locale={locale}
      canManage={canManage}
    />
  );
}

function OrganizationSettingForm({
  organizationId,
  locale,
  canManage,
}: {
  organizationId: number;
  locale: OrganizationSetting;
  canManage: boolean;
}) {
  const t = useT('setting');
  const [value, setValue] = useState(String(locale.value));
  const [clientError, setClientError] = useState<string>();
  const setMutation = useSetOrganizationSetting(organizationId);
  const resetMutation = useResetOrganizationSetting(organizationId);
  const pending = setMutation.isPending || resetMutation.isPending;
  const mutationError = setMutation.error ?? resetMutation.error;

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const input = localeSettingMutationSchema.safeParse({ value });
    if (!input.success) {
      setClientError(t('errors.invalidValue'));
      return;
    }
    setClientError(undefined);
    setMutation.mutate(
      {
        key: 'localization.locale',
        input: input.data,
        expectedVersion: locale.version,
      },
      { onSuccess: () => toast.success(t('messages.saved')) }
    );
  };

  return (
    <section className="py-6" aria-labelledby="organization-setting-heading">
      <div className="mb-5">
        <h2 id="organization-setting-heading" className="text-base font-semibold">
          {t('organization.title')}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">{t('organization.description')}</p>
      </div>

      <form onSubmit={submit} className="max-w-xl space-y-5" aria-busy={pending}>
        {clientError || mutationError ? (
          <Alert variant="destructive">
            <AlertCircle className="size-4" aria-hidden="true" />
            <AlertDescription>
              {clientError ?? t(resolveSettingErrorKey(mutationError))}
            </AlertDescription>
          </Alert>
        ) : null}
        <div className="grid gap-4 sm:grid-cols-[1fr_16rem_auto] sm:items-end">
          <div className="flex min-w-0 gap-3">
            <Globe2 className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <div>
              <Label htmlFor="organization-setting-locale">{t('fields.locale')}</Label>
              <p className="mt-1 text-sm text-muted-foreground">{t('fields.localeDescription')}</p>
            </div>
          </div>
          <Select
            value={value}
            disabled={!canManage || pending}
            onValueChange={next => {
              setValue(next);
              setClientError(undefined);
            }}
          >
            <SelectTrigger id="organization-setting-locale" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en-US">{t('options.enUS')}</SelectItem>
              <SelectItem value="zh-Hans">{t('options.zhHans')}</SelectItem>
            </SelectContent>
          </Select>
          {canManage ? (
            <Button
              type="button"
              isIcon
              variant="ghost"
              title={t('actions.reset')}
              aria-label={t('actions.resetLocale')}
              disabled={pending || locale.source === 'default'}
              onClick={() =>
                resetMutation.mutate(
                  { key: 'localization.locale', expectedVersion: locale.version },
                  { onSuccess: () => toast.success(t('messages.reset')) }
                )
              }
            >
              <RotateCcw className="size-4" />
            </Button>
          ) : null}
        </div>
        {canManage ? (
          <Button
            type="submit"
            icon={<Save className="size-4" />}
            loading={setMutation.isPending}
            disabled={pending || value === locale.value}
          >
            {t('actions.save')}
          </Button>
        ) : null}
      </form>
    </section>
  );
}

function OrganizationSettingSkeleton() {
  return (
    <div className="space-y-5 py-6" aria-hidden="true">
      <Skeleton className="h-14 w-full max-w-xl" />
      <Skeleton className="h-20 w-full max-w-xl" />
    </div>
  );
}
