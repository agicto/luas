import { createFileRoute, redirect } from '@tanstack/react-router';
import { ConsoleLayout } from '@/components/layout/console-layout';
import { isFeatureEnabled } from '@/config/env';
import {
  isSignedOut,
  operatorSessionQueryOptions,
} from '@/features/operator-session/hooks/use-operator-session';

export const Route = createFileRoute('/console')({
  // UX only: the API authorizes every operator request independently.
  beforeLoad: async ({ context, location }) => {
    if (!isFeatureEnabled('operator')) {
      return;
    }
    try {
      await context.queryClient.ensureQueryData(operatorSessionQueryOptions());
    } catch (error) {
      if (isSignedOut(error)) {
        throw redirect({ to: '/login', search: { redirect: location.href }, replace: true });
      }
      throw error;
    }
  },
  component: ConsoleLayout,
});
