import { createFileRoute } from '@tanstack/react-router';
import { ConsoleLayout } from '@/components/layout/console-layout';

export const Route = createFileRoute('/console')({
  component: ConsoleLayout,
});
