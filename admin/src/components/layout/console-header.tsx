import { useRouterState } from '@tanstack/react-router';
import { ChevronRight, Moon, Sun } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { StatusBadge } from '@/components/ui/status-badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useResolvedTheme } from '@/features/preferences/hooks/use-resolved-theme';
import { usePreferencesStore } from '@/features/preferences/store/preferences-store';

export function ConsoleHeader() {
  const { t } = useTranslation();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const dark = useResolvedTheme();
  const setTheme = usePreferencesStore((state) => state.setTheme);
  const currentPage =
    pathname === '/console/preferences' ? t('navigation.preferences') : t('navigation.overview');
  const themeLabel = dark ? t('preferences.light') : t('preferences.dark');

  return (
    <header className="console-header-material sticky top-0 z-20 flex h-14 shrink-0 items-center gap-2 px-4">
      <SidebarTrigger
        label={t('navigation.toggle')}
        aria-label={t('navigation.toggle')}
        title={t('navigation.toggle')}
        className="-ml-1"
      />
      <Separator orientation="vertical" className="mr-1 h-4" />

      <nav
        className="flex min-w-0 items-center gap-1.5 text-sm"
        aria-label={t('navigation.breadcrumb')}
      >
        <span className="hidden text-muted-foreground sm:inline">{t('navigation.console')}</span>
        <ChevronRight
          className="hidden size-3.5 text-muted-foreground sm:block"
          aria-hidden="true"
        />
        <span className="truncate font-medium">{currentPage}</span>
      </nav>

      <div className="ml-auto flex items-center gap-2">
        <StatusBadge className="hidden sm:inline-flex">{t('common.adminConsole')}</StatusBadge>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="size-8"
              aria-label={themeLabel}
              onClick={() => setTheme(dark ? 'light' : 'dark')}
            >
              {dark ? <Sun aria-hidden="true" /> : <Moon aria-hidden="true" />}
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">{themeLabel}</TooltipContent>
        </Tooltip>
      </div>
    </header>
  );
}
