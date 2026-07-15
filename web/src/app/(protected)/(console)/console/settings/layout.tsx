import type { PropsWithChildren } from 'react';
import { getMessages } from 'next-intl/server';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';
import { isWebFeatureEnabled } from '@/config/features';

export default async function SettingsLayout({ children }: PropsWithChildren) {
  const messages = selectMessageNamespaces(
    await getMessages(),
    isWebFeatureEnabled('setting')
      ? [...CLIENT_MESSAGE_NAMESPACES.settings, ...CLIENT_MESSAGE_NAMESPACES.setting]
      : CLIENT_MESSAGE_NAMESPACES.settings
  );
  return <RouteMessagesProvider additionalMessages={messages}>{children}</RouteMessagesProvider>;
}
