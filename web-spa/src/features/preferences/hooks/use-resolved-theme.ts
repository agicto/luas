import { useSyncExternalStore } from 'react';
import {
  type ThemePreference,
  usePreferencesStore,
} from '@/features/preferences/store/preferences-store';

const darkModeQuery = '(prefers-color-scheme: dark)';

function subscribeToSystemTheme(onChange: () => void) {
  const media = window.matchMedia(darkModeQuery);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

function systemThemeSnapshot() {
  return window.matchMedia(darkModeQuery).matches;
}

export function resolveDarkTheme(theme: ThemePreference, systemDark: boolean) {
  return theme === 'dark' || (theme === 'system' && systemDark);
}

export function useResolvedTheme() {
  const theme = usePreferencesStore((state) => state.theme);
  const systemDark = useSyncExternalStore(subscribeToSystemTheme, systemThemeSnapshot, () => false);

  return resolveDarkTheme(theme, systemDark);
}
