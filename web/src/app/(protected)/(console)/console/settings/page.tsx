import { lazy, Suspense } from 'react';
import { KeyRound, SlidersHorizontal } from 'lucide-react';

import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { isWebFeatureEnabled } from '@/config/features';
import { ApiKeyPanel } from '@/features/api-key';
import { getT } from '@/i18n/server';

const UserSettingPanel = lazy(async () => {
  const feature = await import('@/features/setting/components/user-setting-panel');
  return { default: feature.UserSettingPanel };
});

/**
 * Replaceable settings surface for the starter console.
 */
export default async function SettingsPage() {
  const t = await getT('settings');
  const settingEnabled = isWebFeatureEnabled('setting');

  return (
    <div className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6 lg:py-8">
      <h1 className="text-2xl font-semibold">{t('title')}</h1>

      <Tabs defaultValue={settingEnabled ? 'preferences' : 'api'} className="mt-5">
        <TabsList className="max-w-full overflow-x-auto">
          {settingEnabled ? (
            <TabsTrigger value="preferences">
              <SlidersHorizontal aria-hidden="true" />
              {t('tabs.preferences')}
            </TabsTrigger>
          ) : null}
          <TabsTrigger value="api">
            <KeyRound aria-hidden="true" />
            {t('tabs.api')}
          </TabsTrigger>
        </TabsList>

        {settingEnabled ? (
          <TabsContent value="preferences" className="pt-5">
            <Suspense fallback={<UserSettingPanelFallback />}>
              <UserSettingPanel />
            </Suspense>
          </TabsContent>
        ) : null}

        <TabsContent value="api" className="pt-5">
          <ApiKeyPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function UserSettingPanelFallback() {
  return (
    <div className="max-w-2xl space-y-5 py-2" aria-hidden="true">
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-20 w-full" />
      <Skeleton className="h-20 w-full" />
    </div>
  );
}
