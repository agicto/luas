import { useState } from 'react';
import { Link, Outlet } from '@tanstack/react-router';
import { Gauge, Menu, Moon, Settings2, Sun, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { env } from '@/config/env';
import { usePreferencesStore } from '@/features/preferences/store/preferences-store';
import { cn } from '@/lib/cn';

export function ConsoleLayout() {
  const { t } = useTranslation();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const theme = usePreferencesStore((state) => state.theme);
  const setTheme = usePreferencesStore((state) => state.setTheme);
  const dark = theme === 'dark';

  const navigation = [
    {
      icon: Gauge,
      label: t('navigation.overview'),
      to: '/console',
      exact: true,
    },
    {
      icon: Settings2,
      label: t('navigation.preferences'),
      to: '/console/preferences',
      exact: false,
    },
  ] as const;

  return (
    <div className="app-frame">
      {navigationOpen ? (
        <button
          className="navigation-backdrop"
          aria-label={t('navigation.close')}
          onClick={() => setNavigationOpen(false)}
        />
      ) : null}

      <aside className={cn('sidebar', navigationOpen && 'sidebar-open')} id="primary-navigation">
        <div className="sidebar-brand">
          <span className="brand-mark" aria-hidden="true">
            L
          </span>
          <div className="min-w-0">
            <strong>{env.APP_NAME}</strong>
            <span>{t('app.shell')}</span>
          </div>
          <Button
            className="ml-auto lg:hidden"
            aria-label={t('navigation.close')}
            title={t('navigation.close')}
            size="icon"
            variant="ghost"
            onClick={() => setNavigationOpen(false)}
          >
            <X className="size-4" />
          </Button>
        </div>

        <nav className="sidebar-nav" aria-label={t('app.shell')}>
          {navigation.map((item) => {
            const Icon = item.icon;
            return (
              <Link
                key={item.to}
                to={item.to}
                activeOptions={{ exact: item.exact }}
                className="navigation-link"
                activeProps={{ className: 'navigation-link-active' }}
                onClick={() => setNavigationOpen(false)}
              >
                <Icon className="size-4" aria-hidden="true" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="sidebar-footer">
          <StatusBadge>{t('common.staticSpa')}</StatusBadge>
          <span>v0.1</span>
        </div>
      </aside>

      <div className="app-main">
        <header className="mobile-header">
          <Button
            aria-controls="primary-navigation"
            aria-expanded={navigationOpen}
            aria-label={t('navigation.open')}
            title={t('navigation.open')}
            size="icon"
            variant="ghost"
            onClick={() => setNavigationOpen(true)}
          >
            <Menu className="size-5" />
          </Button>
          <strong>{env.APP_NAME}</strong>
          <Button
            aria-label={dark ? t('preferences.light') : t('preferences.dark')}
            title={dark ? t('preferences.light') : t('preferences.dark')}
            size="icon"
            variant="ghost"
            onClick={() => setTheme(dark ? 'light' : 'dark')}
          >
            {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>
        </header>

        <main className="page-container">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
