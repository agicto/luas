import { notFound } from 'next/navigation';

import { isWebFeatureEnabled } from '@/config/features';
import { AssetPanel } from '@/features/asset/components/asset-panel';

export default function AssetsPage() {
  if (!isWebFeatureEnabled('asset')) notFound();
  return <AssetPanel />;
}
