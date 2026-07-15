import type { PropsWithChildren } from 'react';
import { getMessages } from 'next-intl/server';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';
import { isWebFeatureEnabled } from '@/config/features';

export default async function OrganizationRouteLayout({ children }: PropsWithChildren) {
  const messages = selectMessageNamespaces(
    await getMessages(),
    isWebFeatureEnabled('permission')
      ? [
          ...CLIENT_MESSAGE_NAMESPACES.organization,
          ...CLIENT_MESSAGE_NAMESPACES.permission,
        ]
      : CLIENT_MESSAGE_NAMESPACES.organization
  );

  return (
    <RouteMessagesProvider additionalMessages={messages}>
      {children}
    </RouteMessagesProvider>
  );
}
