import { notFound } from 'next/navigation';

import { isWebFeatureEnabled } from '@/config/features';
import { UserUsagePanel } from '@/features/usage/components/usage-panel';
import { getT } from '@/i18n/server';

export default async function UsagePage() {
  if (!isWebFeatureEnabled('usage')) notFound();
  const t = await getT('usage');

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:py-8">
      <div className="border-b pb-5">
        <h1 className="text-2xl font-semibold">{t('title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('description')}</p>
      </div>
      <div className="pt-5">
        <UserUsagePanel />
      </div>
    </div>
  );
}
