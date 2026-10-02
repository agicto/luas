import { createFileRoute, notFound, redirect } from '@tanstack/react-router';
import { isOperatorFeatureEnabled } from '@/config/env';
import { OrganizationDetailPage } from '@/features/organizations/components/organization-detail-page';
import { WebhookDiagnostics } from '@/features/webhooks/components/webhook-diagnostics';

export const Route = createFileRoute('/console/organizations/$organizationId')({
  params: {
    parse: ({ organizationId }) => {
      if (!/^[1-9][0-9]{0,15}$/.test(organizationId)) {
        throw notFound();
      }
      return { organizationId };
    },
  },
  beforeLoad: () => {
    if (!isOperatorFeatureEnabled('organization')) {
      throw redirect({ to: '/console', replace: true });
    }
  },
  component: function OrganizationRoute() {
    const organizationId = Number(Route.useParams().organizationId);
    const navigate = Route.useNavigate();
    return (
      <OrganizationDetailPage
        key={organizationId}
        organizationId={organizationId}
        onBack={() => void navigate({ to: '/console/organizations' })}
      >
        {isOperatorFeatureEnabled('webhook') ? (
          <WebhookDiagnostics organizationId={organizationId} />
        ) : null}
      </OrganizationDetailPage>
    );
  },
});
