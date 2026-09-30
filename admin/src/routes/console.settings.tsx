import { createFileRoute, redirect } from '@tanstack/react-router';
import { isFeatureEnabled } from '@/config/env';
import { AppSettingsPage } from '@/features/app-settings/components/app-settings-page';

export const Route = createFileRoute('/console/settings')({
  beforeLoad: () => {
    if (!isFeatureEnabled('operator')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: AppSettingsPage,
});
