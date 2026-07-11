'use client';

import { useMemo, type PropsWithChildren } from 'react';
import { NextIntlClientProvider, useLocale, useMessages } from 'next-intl';

import type { Messages } from './modules';

interface RouteMessagesProviderProps extends PropsWithChildren {
  additionalMessages: Partial<Messages>;
}

/** Adds route-owned top-level namespaces without discarding the global client messages. */
export function RouteMessagesProvider({
  additionalMessages,
  children,
}: RouteMessagesProviderProps) {
  const locale = useLocale();
  const inheritedMessages = useMessages();
  const messages = useMemo(
    () => ({ ...inheritedMessages, ...additionalMessages }),
    [additionalMessages, inheritedMessages]
  );

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      {children}
    </NextIntlClientProvider>
  );
}
