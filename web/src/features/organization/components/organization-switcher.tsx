'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Building2, Check, ChevronsUpDown } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ROUTES, getOrganizationRoute } from '@/constants/routes';
import { useOrganizations } from '@/features/organization/hooks/use-organizations';
import { useT } from '@/i18n';

const organizationPath = /^\/console\/organizations\/([1-9]\d*)(?:\/|$)/;

export function OrganizationSwitcher() {
  const pathname = usePathname();
  const t = useT();
  const organizations = useOrganizations();
  const match = organizationPath.exec(pathname);
  const selectedId = match ? Number(match[1]) : null;
  const selected = organizations.data?.items.find(({ id }) => id === selectedId);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          noScale
          className="max-w-[11rem] justify-between md:max-w-[15rem]"
          aria-label={t('nav.organizations')}
        >
          <Building2 className="size-4" aria-hidden="true" />
          <span className="hidden min-w-0 flex-1 truncate text-left sm:block">
            {selected?.name ?? t('nav.organizations')}
          </span>
          <ChevronsUpDown className="hidden size-3.5 text-muted-foreground sm:block" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        <DropdownMenuLabel>{t('nav.organizations')}</DropdownMenuLabel>
        {organizations.data?.items.map((organization) => (
          <DropdownMenuItem key={organization.id} asChild>
            <Link href={getOrganizationRoute(organization.id)}>
              <span className="min-w-0 flex-1 truncate">{organization.name}</span>
              {organization.id === selectedId ? (
                <Check className="ml-auto size-4" aria-hidden="true" />
              ) : null}
            </Link>
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link href={ROUTES.CONSOLE.ORGANIZATIONS}>
            <Building2 className="size-4" aria-hidden="true" />
            {t('nav.organizations')}
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
