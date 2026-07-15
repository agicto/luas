import type { PropsWithChildren } from 'react';
import { getMessages } from 'next-intl/server';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';

export default async function SettingsLayout({ children }: PropsWithChildren) {
  const messages = selectMessageNamespaces(await getMessages(), CLIENT_MESSAGE_NAMESPACES.settings);
  return <RouteMessagesProvider additionalMessages={messages}>{children}</RouteMessagesProvider>;
}
