import { Link, useRouterState } from '@tanstack/react-router';
import { Gauge, PanelsTopLeft, Settings2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from '@/components/ui/sidebar';
import { env } from '@/config/env';

const navigationItems = [
  {
    icon: Gauge,
    labelKey: 'navigation.overview',
    to: '/console',
  },
  {
    icon: Settings2,
    labelKey: 'navigation.preferences',
    to: '/console/preferences',
  },
] as const;

export function AppSidebar() {
  const { t } = useTranslation();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const { setOpenMobile } = useSidebar();

  return (
    <Sidebar
      variant="inset"
      collapsible="icon"
      mobileTitle={t('navigation.console')}
      mobileDescription={t('navigation.mobileDescription')}
    >
      <SidebarHeader className="border-b border-sidebar-border">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              size="lg"
              tooltip={env.APP_NAME}
              className="h-12 data-[active=true]:bg-transparent"
              isActive={pathname === '/console'}
            >
              <Link to="/console" onClick={() => setOpenMobile(false)}>
                <span className="brand-mark" aria-hidden="true">
                  L
                </span>
                <span className="grid min-w-0 flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{env.APP_NAME}</span>
                  <span className="truncate text-xs text-muted-foreground">{t('app.shell')}</span>
                </span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{t('navigation.section')}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navigationItems.map((item) => {
                const Icon = item.icon;
                const label = t(item.labelKey);
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton asChild isActive={pathname === item.to} tooltip={label}>
                      <Link to={item.to} onClick={() => setOpenMobile(false)}>
                        <Icon aria-hidden="true" />
                        <span>{label}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border">
        <div className="flex min-w-0 items-center gap-2 px-2 py-1.5">
          <span className="grid size-8 shrink-0 place-items-center rounded-md border border-sidebar-border bg-background text-muted-foreground">
            <PanelsTopLeft className="size-4" aria-hidden="true" />
          </span>
          <span className="grid min-w-0 flex-1 text-xs leading-tight group-data-[collapsible=icon]:hidden">
            <span className="truncate font-medium text-sidebar-foreground">
              {t('common.staticSpa')}
            </span>
            <span className="truncate text-muted-foreground">v0.1</span>
          </span>
        </div>
      </SidebarFooter>

      <SidebarRail aria-label={t('navigation.toggle')} title={t('navigation.toggle')} />
    </Sidebar>
  );
}
