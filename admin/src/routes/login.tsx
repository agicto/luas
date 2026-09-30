import { createFileRoute, redirect } from '@tanstack/react-router';
import { z } from 'zod';
import { isFeatureEnabled } from '@/config/env';
import { SignInPage } from '@/features/operator-session/components/sign-in-page';

// Only console paths are accepted so the sign-in page cannot become an open redirect.
const searchSchema = z.object({
  redirect: z
    .string()
    .regex(/^\/console(?:[/?#]|$)/)
    .optional()
    .catch(undefined),
});

export const Route = createFileRoute('/login')({
  validateSearch: searchSchema,
  beforeLoad: () => {
    if (!isFeatureEnabled('operator')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function LoginRoute() {
    const { redirect } = Route.useSearch();
    return <SignInPage redirectTo={redirect ?? '/console'} />;
  },
});
