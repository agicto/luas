import { useEffect } from 'react';
import { useResolvedTheme } from '@/features/preferences/hooks/use-resolved-theme';

export function ThemeSync() {
  const dark = useResolvedTheme();

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark);
    document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', dark ? '#171717' : '#fafafa');
  }, [dark]);

  return null;
}
