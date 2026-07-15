'use client';

import { useId, useState } from 'react';
import { AlertCircle, Clock3, Globe2, RefreshCw, RotateCcw, Save } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
  useResetUserSetting,
  useSetUserSetting,
  useUserSettings,
} from '@/features/setting/hooks/use-settings';
import {
  localeSettingMutationSchema,
  timezoneSettingMutationSchema,
} from '@/features/setting/schemas';
import type { UserSetting } from '@/features/setting/types';
import { resolveSettingErrorKey } from '@/features/setting/utils/setting-error';
import { useT } from '@/i18n';

const commonTimezones = [
  'UTC',
  'Africa/Johannesburg',
  'America/Chicago',
  'America/Los_Angeles',
  'America/New_York',
  'America/Sao_Paulo',
  'Asia/Dubai',
  'Asia/Hong_Kong',
  'Asia/Shanghai',
  'Asia/Singapore',
  'Asia/Tokyo',
  'Australia/Sydney',
  'Europe/Berlin',
  'Europe/Dublin',
  'Europe/London',
  'Europe/Paris',
] as const;

export function UserSettingPanel() {
  const t = useT('setting');
  const query = useUserSettings();

  if (query.isPending) return <SettingPanelSkeleton />;
  if (query.error || !query.data) {
    return (
      <Alert variant="destructive">
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

  const locale = query.data.find(item => item.key === 'localization.locale');
  const timezone = query.data.find(item => item.key === 'localization.timezone');
  if (!locale || !timezone) return null;

  return (
    <UserSettingForm
      key={`${locale.version}:${timezone.version}`}
      locale={locale}
      timezone={timezone}
    />
  );
}

function UserSettingForm({ locale, timezone }: { locale: UserSetting; timezone: UserSetting }) {
  const t = useT('setting');
  const timezoneListId = useId();
  const [localeValue, setLocaleValue] = useState(String(locale.value));
  const [timezoneValue, setTimezoneValue] = useState(String(timezone.value));
  const [clientError, setClientError] = useState<string>();
  const setMutation = useSetUserSetting();
  const resetMutation = useResetUserSetting();
  const pending = setMutation.isPending || resetMutation.isPending;
  const changed = localeValue !== locale.value || timezoneValue !== timezone.value;
  const mutationError = setMutation.error ?? resetMutation.error;

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const localeInput = localeSettingMutationSchema.safeParse({ value: localeValue });
    const timezoneInput = timezoneSettingMutationSchema.safeParse({ value: timezoneValue });
    if (!localeInput.success || !timezoneInput.success) {
      setClientError(t('errors.invalidValue'));
      return;
    }
    setClientError(undefined);
    try {
      if (localeValue !== locale.value) {
        await setMutation.mutateAsync({
          key: 'localization.locale',
          input: localeInput.data,
          expectedVersion: locale.version,
        });
      }
      if (timezoneValue !== timezone.value) {
        await setMutation.mutateAsync({
          key: 'localization.timezone',
          input: timezoneInput.data,
          expectedVersion: timezone.version,
        });
      }
      toast.success(t('messages.saved'));
    } catch {
      // The reviewed error surface below owns user-facing copy.
    }
  };

  const reset = (setting: UserSetting) => {
    resetMutation.mutate(
      { key: setting.key, expectedVersion: setting.version },
      { onSuccess: () => toast.success(t('messages.reset')) }
    );
  };

  return (
    <section className="max-w-2xl py-2" aria-labelledby="user-setting-heading">
      <div className="border-b pb-5">
        <h2 id="user-setting-heading" className="text-base font-semibold">
          {t('user.title')}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">{t('user.description')}</p>
      </div>

      <form onSubmit={handleSubmit} className="divide-y" aria-busy={pending}>
        {clientError || mutationError ? (
          <Alert variant="destructive" className="mt-5">
            <AlertCircle className="size-4" aria-hidden="true" />
            <AlertDescription>
              {clientError ?? t(resolveSettingErrorKey(mutationError))}
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="grid gap-4 py-5 sm:grid-cols-[1fr_16rem_auto] sm:items-end">
          <div className="flex min-w-0 gap-3">
            <Globe2 className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <div>
              <Label htmlFor="user-setting-locale">{t('fields.locale')}</Label>
              <p className="mt-1 text-sm text-muted-foreground">{t('fields.localeDescription')}</p>
            </div>
          </div>
          <Select
            value={localeValue}
            disabled={pending}
            onValueChange={value => {
              setLocaleValue(value);
              setClientError(undefined);
            }}
          >
            <SelectTrigger id="user-setting-locale" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en-US">{t('options.enUS')}</SelectItem>
              <SelectItem value="zh-Hans">{t('options.zhHans')}</SelectItem>
            </SelectContent>
          </Select>
          <Button
            type="button"
            isIcon
            variant="ghost"
            title={t('actions.reset')}
            aria-label={t('actions.resetLocale')}
            disabled={pending || locale.source === 'default'}
            onClick={() => reset(locale)}
          >
            <RotateCcw className="size-4" />
          </Button>
        </div>

        <div className="grid gap-4 py-5 sm:grid-cols-[1fr_16rem_auto] sm:items-end">
          <div className="flex min-w-0 gap-3">
            <Clock3 className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            <div>
              <Label htmlFor="user-setting-timezone">{t('fields.timezone')}</Label>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('fields.timezoneDescription')}
              </p>
            </div>
          </div>
          <div>
            <Input
              id="user-setting-timezone"
              list={timezoneListId}
              value={timezoneValue}
              maxLength={64}
              disabled={pending}
              onChange={event => {
                setTimezoneValue(event.target.value);
                setClientError(undefined);
              }}
            />
            <datalist id={timezoneListId}>
              {commonTimezones.map(value => (
                <option key={value} value={value} />
              ))}
            </datalist>
          </div>
          <Button
            type="button"
            isIcon
            variant="ghost"
            title={t('actions.reset')}
            aria-label={t('actions.resetTimezone')}
            disabled={pending || timezone.source === 'default'}
            onClick={() => reset(timezone)}
          >
            <RotateCcw className="size-4" />
          </Button>
        </div>

        <div className="flex justify-end pt-5">
          <Button
            type="submit"
            icon={<Save className="size-4" />}
            loading={setMutation.isPending}
            disabled={pending || !changed}
          >
            {t('actions.save')}
          </Button>
        </div>
      </form>
    </section>
  );
}

function SettingPanelSkeleton() {
  return (
    <div className="max-w-2xl space-y-5 py-2" aria-hidden="true">
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-20 w-full" />
      <Skeleton className="h-20 w-full" />
    </div>
  );
}
