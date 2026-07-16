import { act } from 'react';
import { hydrateRoot, type Root } from 'react-dom/client';
import { renderToString } from 'react-dom/server';
import { fireEvent, render, screen } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ThemeToggle } from '@/components/theme-toggle';
import commonMessages from '@/i18n/modules/common/en-US';

const themeState = vi.hoisted(() => ({
  selected: 'system',
  setTheme: vi.fn(),
}));

vi.mock('next-themes', () => ({
  useTheme: () => ({
    theme: themeState.selected,
    setTheme: themeState.setTheme,
  }),
}));

function renderThemeToggle() {
  return (
    <NextIntlClientProvider
      locale="en-US"
      messages={{ common: commonMessages } as unknown as AbstractIntlMessages}
    >
      <ThemeToggle />
    </NextIntlClientProvider>
  );
}

describe('ThemeToggle', () => {
  beforeEach(() => {
    themeState.selected = 'system';
    themeState.setTheme.mockClear();
  });

  it('uses one native three-state selector without a custom menu runtime', () => {
    render(renderThemeToggle());

    const selector = screen.getByRole('combobox', { name: 'Toggle theme' });
    expect(selector).toHaveValue('system');
    expect(screen.getAllByRole('option').map(option => option.textContent)).toEqual([
      'Light',
      'Dark',
      'System',
    ]);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();

    fireEvent.change(selector, { target: { value: 'dark' } });
    expect(themeState.setTheme).toHaveBeenCalledWith('dark');
  });

  it('hydrates a persisted theme without changing the server first frame', async () => {
    themeState.selected = 'dark';
    const container = document.createElement('div');
    const serverMarkup = renderToString(renderThemeToggle());
    expect(serverMarkup).toContain('<option value="system" selected="">System</option>');
    container.innerHTML = serverMarkup;
    document.body.appendChild(container);

    expect(container.querySelector('option[value="system"]')).toHaveAttribute('selected');

    const recoverableError = vi.fn();
    let root: Root | undefined;
    await act(async () => {
      root = hydrateRoot(container, renderThemeToggle(), { onRecoverableError: recoverableError });
    });

    expect(recoverableError).not.toHaveBeenCalled();
    expect(container.querySelector('select')).toHaveValue('dark');

    await act(async () => root?.unmount());
    container.remove();
  });
});
