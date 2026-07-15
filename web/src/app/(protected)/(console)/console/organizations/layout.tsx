import type { PropsWithChildren } from 'react';
import { getMessages } from 'next-intl/server';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';
import { isWebFeatureEnabled } from '@/config/features';

export default async function OrganizationRouteLayout({ children }: PropsWithChildren) {
  const namespaces = [
    ...CLIENT_MESSAGE_NAMESPACES.organization,
    ...(isWebFeatureEnabled('permission') ? CLIENT_MESSAGE_NAMESPACES.permission : []),
    ...(isWebFeatureEnabled('setting') ? CLIENT_MESSAGE_NAMESPACES.setting : []),
  ];
  const messages = selectMessageNamespaces(await getMessages(), namespaces);

  return <RouteMessagesProvider additionalMessages={messages}>{children}</RouteMessagesProvider>;
}
