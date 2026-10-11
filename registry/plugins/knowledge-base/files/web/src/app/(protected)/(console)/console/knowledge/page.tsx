import { Suspense } from 'react';
import { notFound } from 'next/navigation';

import { Skeleton } from '@/components/ui/skeleton';
import { isWebFeatureEnabled } from '@/config/features';
import { KnowledgeWorkspace } from '@/features/knowledge/components/knowledge-workspace';
import { getT } from '@/i18n/server';

export default async function KnowledgePage() {
  if (!isWebFeatureEnabled('knowledge')) notFound();
  const t = await getT('knowledge');

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:py-8">
      <div className="border-b pb-5">
        <h1 className="text-2xl font-semibold">{t('title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('description')}</p>
      </div>
      <div className="pt-5">
        <Suspense fallback={<Skeleton className="h-48 w-full" />}>
          <KnowledgeWorkspace />
        </Suspense>
      </div>
    </div>
  );
}
