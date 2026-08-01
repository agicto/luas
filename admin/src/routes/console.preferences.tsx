import { createFileRoute } from '@tanstack/react-router';
import { PreferencesPage } from '@/features/preferences/components/preferences-page';

export const Route = createFileRoute('/console/preferences')({
  component: PreferencesPage,
});
