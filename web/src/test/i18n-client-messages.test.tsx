import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, screen } from '@testing-library/react';
import { NextIntlClientProvider, useTranslations } from 'next-intl';
import { describe, expect, it } from 'vitest';

import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { messages } from '@/i18n/modules';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';

const sourceRoot = resolve(process.cwd(), 'src');

function readSource(path: string): string {
  return readFileSync(resolve(sourceRoot, path), 'utf8');
}

function TranslationProbe() {
  const t = useTranslations();

  return <p>{`${t('common.save')} / ${t('auth.login')}`}</p>;
}

describe('client i18n message boundaries', () => {
  it('keeps global and route-owned namespaces explicit', () => {
    expect(CLIENT_MESSAGE_NAMESPACES).toEqual({
      global: ['common', 'errors'],
      auth: ['auth'],
      console: ['auth', 'nav', 'console'],
      settings: ['settings'],
      organization: ['organization'],
      permission: ['permission'],
      notification: ['notification'],
      asset: ['asset'],
      setting: ['setting'],
      usage: ['usage'],
      i18nTest: ['test'],
    });
  });

  it('selects only the requested top-level namespaces', () => {
    const selected = selectMessageNamespaces(messages, ['errors', 'common']);

    expect(Object.keys(selected)).toEqual(['errors', 'common']);
    expect(selected).toEqual({
      errors: messages.errors,
      common: messages.common,
    });
  });

  it('merges route-owned messages with the inherited client context', () => {
    render(
      <NextIntlClientProvider
        locale="en-US"
        messages={selectMessageNamespaces(messages, CLIENT_MESSAGE_NAMESPACES.global)}
      >
        <RouteMessagesProvider
          additionalMessages={selectMessageNamespaces(messages, CLIENT_MESSAGE_NAMESPACES.auth)}
        >
          <TranslationProbe />
        </RouteMessagesProvider>
      </NextIntlClientProvider>
    );

    expect(screen.getByText('Save / Sign In')).toBeInTheDocument();
  });

  it('wires each client message scope at its owning route boundary', () => {
    const rootLayout = readSource('app/layout.tsx');
    const authLayout = readSource('app/(auth)/layout.tsx');
    const consoleLayout = readSource('app/(protected)/(console)/layout.tsx');
    const organizationLayout = readSource(
      'app/(protected)/(console)/console/organizations/layout.tsx'
    );
    const settingsLayout = readSource('app/(protected)/(console)/console/settings/layout.tsx');
    const usageLayout = readSource('app/(protected)/(console)/console/usage/layout.tsx');
    const i18nTestLayout = readSource('app/(protected)/(devtools)/i18n-test/layout.tsx');

    expect(rootLayout).toContain('CLIENT_MESSAGE_NAMESPACES.global');
    expect(rootLayout).not.toContain('messages={messages}');
    expect(authLayout).toContain('CLIENT_MESSAGE_NAMESPACES.auth');
    expect(consoleLayout).toContain('CLIENT_MESSAGE_NAMESPACES.console');
    expect(consoleLayout).toContain('CLIENT_MESSAGE_NAMESPACES.notification');
    expect(consoleLayout).toContain('CLIENT_MESSAGE_NAMESPACES.asset');
    expect(settingsLayout).toContain('CLIENT_MESSAGE_NAMESPACES.settings');
    expect(settingsLayout).toContain('CLIENT_MESSAGE_NAMESPACES.setting');
    expect(organizationLayout).toContain('CLIENT_MESSAGE_NAMESPACES.organization');
    expect(organizationLayout).toContain('CLIENT_MESSAGE_NAMESPACES.setting');
    expect(organizationLayout).toContain('CLIENT_MESSAGE_NAMESPACES.usage');
    expect(usageLayout).toContain('CLIENT_MESSAGE_NAMESPACES.usage');
    expect(i18nTestLayout).toContain('CLIENT_MESSAGE_NAMESPACES.i18nTest');
  });
});
