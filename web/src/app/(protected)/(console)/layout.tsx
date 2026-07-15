import type { PropsWithChildren } from 'react';
import { getMessages } from 'next-intl/server';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';
import { isWebFeatureEnabled } from '@/config/features';

export default async function ConsoleRouteGroupLayout({ children }: PropsWithChildren) {
  const namespaces = [
    ...CLIENT_MESSAGE_NAMESPACES.console,
    ...(isWebFeatureEnabled('notification') ? CLIENT_MESSAGE_NAMESPACES.notification : []),
    ...(isWebFeatureEnabled('asset') ? CLIENT_MESSAGE_NAMESPACES.asset : []),
  ];
  const messages = selectMessageNamespaces(await getMessages(), namespaces);

  return <RouteMessagesProvider additionalMessages={messages}>{children}</RouteMessagesProvider>;
}
