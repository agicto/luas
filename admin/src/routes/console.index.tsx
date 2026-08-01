import { createFileRoute } from '@tanstack/react-router';
import { OverviewPage } from '@/features/system/components/overview-page';

export const Route = createFileRoute('/console/')({
  component: OverviewPage,
});
