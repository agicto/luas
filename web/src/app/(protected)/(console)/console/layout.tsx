'use client';

import { lazy, Suspense } from 'react';
import Link from "next/link";
import { usePathname } from "next/navigation";
import { LucideIcon, BarChart3, Settings, Home, Bell, LogOut, Palette, Building2 } from "lucide-react";

import { cn } from "@/utils";
import { ROUTES } from "@/constants/routes";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { ThemeToggle } from "@/components/theme-toggle";
import { LanguageSwitcher } from "@/components/common";
import { useT, type AllTranslationKeys } from "@/i18n";
import { useLogout } from "@/features/auth/hooks/use-auth";
import { useAuthStore } from "@/features/auth/store/auth-store";
import { isWebFeatureEnabled } from "@/config/features";

const OrganizationSwitcher = lazy(async () => {
  const organizationFeature = await import(
    '@/features/organization/components/organization-switcher'
  );
  return { default: organizationFeature.OrganizationSwitcher };
});

interface NavItem {
  titleKey: Extract<AllTranslationKeys, `nav.${string}`>;
  href: string;
  icon: LucideIcon;
}

export default function ConsoleLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const t = useT();
  const user = useAuthStore.use.user();
  const { mutate: logout, isPending: isLoggingOut } = useLogout();
  const organizationEnabled = isWebFeatureEnabled('organization');

  const mainNavItems: NavItem[] = [
    {
      titleKey: "nav.console",
      href: ROUTES.CONSOLE.HOME,
      icon: Home,
    },
    ...(organizationEnabled
      ? [{
          titleKey: "nav.organizations" as const,
          href: ROUTES.CONSOLE.ORGANIZATIONS,
          icon: Building2,
        }]
      : []),
  ];

  const secondaryNavItems: NavItem[] = [
    {
      titleKey: "nav.styleguide",
      href: ROUTES.DEVTOOLS.STYLEGUIDE,
      icon: Palette,
    },
    {
      titleKey: "nav.settings",
      href: ROUTES.CONSOLE.SETTINGS,
      icon: Settings,
    },
  ];

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-bg-canvas text-text-main">
      {/* Fixed Header */}
      <header className="flex h-16 shrink-0 items-center justify-between border-b bg-bg-surface px-4 md:px-6 shadow-sm z-50">
        <div className="flex min-w-0 items-center gap-2 sm:gap-4">
          <Link href={ROUTES.SITE.HOME} className="flex items-center gap-2 group">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-transform group-hover:scale-110">
              <BarChart3 className="h-5 w-5" />
            </div>
            <span className="hidden text-xl font-bold sm:inline">Luas Console</span>
          </Link>
          {organizationEnabled ? (
            <Suspense fallback={<div className="h-8 w-9 rounded-md border bg-muted" aria-hidden="true" />}>
              <OrganizationSwitcher />
            </Suspense>
          ) : null}
        </div>

        <div className="flex shrink-0 items-center gap-1 sm:gap-2">
          <div className="hidden sm:block"><LanguageSwitcher /></div>
          <ThemeToggle />
          
          <Button variant="ghost" isIcon className="relative hidden h-9 w-9 rounded-full sm:inline-flex">
            <Bell className="h-4 w-4 text-text-muted" />
            <span className="absolute top-2 right-2 h-2 w-2 rounded-full bg-primary border-2 border-bg-surface" />
            <span className="sr-only">{t('console.notifications')}</span>
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                isIcon
                noScale
                className="h-9 w-9 rounded-full overflow-hidden border border-border/50 hover:border-primary/50 transition-colors"
              >
                <Avatar className="h-full w-full">
                  <AvatarImage src="https://github.com/shadcn.png" alt="" />
                  <AvatarFallback className="bg-primary/10 text-primary">
                    {user?.name
                      ?.split(' ')
                      .map((part) => part[0])
                      .join('')
                      .slice(0, 2)
                      .toUpperCase() || 'LF'}
                  </AvatarFallback>
                </Avatar>
                <span className="sr-only">{t('console.profile')}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56 rounded-xl p-1 shadow-premium">
              <div className="px-2 py-1.5 text-xs font-medium text-text-muted uppercase tracking-wider">
                {user?.name || t('nav.profile')}
              </div>
              <DropdownMenuItem className="rounded-lg cursor-pointer">
                {t('nav.profile')}
              </DropdownMenuItem>
              <DropdownMenuItem className="rounded-lg cursor-pointer">
                {t('nav.settings')}
              </DropdownMenuItem>
              <div className="h-px bg-border/50 my-1" />
              <DropdownMenuItem
                className="rounded-lg cursor-pointer text-error focus:bg-destructive/10"
                onSelect={(event) => {
                  event.preventDefault();
                  logout();
                }}
                disabled={isLoggingOut}
              >
                <div className="flex w-full items-center">
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>{t('auth.logout')}</span>
                </div>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      {/* Body: Sidebar + Main Content */}
      <div className="flex h-0 grow overflow-hidden">
        {/* Fixed Sidebar */}
        <aside className="hidden w-[220px] shrink-0 border-r bg-background md:flex md:flex-col">
          <div className="flex flex-1 flex-col overflow-y-auto py-2">
            <nav className="grid items-start px-2 text-sm font-medium">
              {mainNavItems.map((item) => {
                const IconComponent = item.icon;
                const isActive = isNavigationItemActive(pathname, item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      "flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary",
                      isActive
                        ? "bg-muted text-primary"
                        : "text-text-subtle hover:bg-muted/50"
                    )}
                  >
                    <IconComponent className="h-4.5 w-4.5" />
                    {t(item.titleKey)}
                  </Link>
                );
              })}
            </nav>
          </div>
          <div className="mt-auto p-4">
            <nav className="grid items-start gap-1 text-sm font-medium">
              {secondaryNavItems.map((item) => {
                const IconComponent = item.icon;
                const isActive = isNavigationItemActive(pathname, item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      "flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary",
                      isActive
                        ? "bg-muted text-primary"
                        : "text-text-subtle hover:bg-muted/50"
                    )}
                  >
                    <IconComponent className="h-4.5 w-4.5" />
                    {t(item.titleKey)}
                  </Link>
                );
              })}
            </nav>
            <Link
              href={ROUTES.SITE.HOME}
              className="flex h-8 items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
            >
              <span>{t('console.returnToSite')}</span>
            </Link>
          </div>
        </aside>

        {/* Scrollable Main Content */}
        <main className="h-full min-w-0 w-full overflow-y-auto">
          {children}
        </main>
      </div>
    </div>
  );
}

function isNavigationItemActive(pathname: string, href: string): boolean {
  return pathname === href || (
    href !== ROUTES.CONSOLE.HOME && pathname.startsWith(`${href}/`)
  );
}
