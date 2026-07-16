'use client';

import { useSyncExternalStore } from 'react';
import { Monitor, Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';

import { useT } from '@/i18n';

const supportedThemes = ['light', 'dark', 'system'] as const;
type SupportedTheme = (typeof supportedThemes)[number];

function isSupportedTheme(theme: string | undefined): theme is SupportedTheme {
  return supportedThemes.some(candidate => candidate === theme);
}

const subscribeToHydration = () => () => undefined;
const getClientHydrationSnapshot = () => true;
const getServerHydrationSnapshot = () => false;

/**
 * Theme toggle component that allows switching between light and dark modes
 */
export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const t = useT('common');
  const hydrated = useSyncExternalStore(
    subscribeToHydration,
    getClientHydrationSnapshot,
    getServerHydrationSnapshot
  );
  const selectedTheme = hydrated && isSupportedTheme(theme) ? theme : 'system';
  const ThemeIcon = selectedTheme === 'light' ? Sun : selectedTheme === 'dark' ? Moon : Monitor;

  return (
    <label
      className="relative inline-flex size-9 shrink-0 rounded-lg focus-within:ring-[3px] focus-within:ring-ring/50"
      title={t('toggleTheme')}
    >
      <select
        aria-label={t('toggleTheme')}
        className="peer absolute inset-0 z-10 size-full cursor-pointer appearance-none opacity-0"
        data-performance-interaction="theme-selector"
        value={selectedTheme}
        onChange={event => setTheme(event.currentTarget.value)}
      >
        <option value="light">{t('themeLight')}</option>
        <option value="dark">{t('themeDark')}</option>
        <option value="system">{t('themeSystem')}</option>
      </select>
      <span
        aria-hidden="true"
        className="pointer-events-none inline-flex size-9 items-center justify-center rounded-lg border bg-background shadow-xs transition-colors peer-hover:bg-accent peer-hover:text-accent-foreground dark:border-input dark:bg-input/30"
      >
        <ThemeIcon className="size-[1.2rem]" />
      </span>
    </label>
  );
}
