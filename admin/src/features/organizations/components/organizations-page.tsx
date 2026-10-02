import { Building2, Search } from 'lucide-react';
import { type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pager } from '@/components/layout/pager';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { useOrganizations } from '@/features/organizations/hooks/use-organizations';
import type { OrganizationSearch } from '@/features/organizations/types';

interface OrganizationsPageProps {
  search: OrganizationSearch;
  onSearchChange: (search: OrganizationSearch) => void;
  onOpen: (organizationId: number) => void;
}

export function OrganizationsPage({ search, onSearchChange, onOpen }: OrganizationsPageProps) {
  const { t, i18n } = useTranslation();
  const organizations = useOrganizations(search);
  const [query, setQuery] = useState(search.q ?? '');
  const meta = organizations.data?.meta;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, { dateStyle: 'medium' });

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSearchChange({ q: query.trim() || undefined, page: undefined });
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('organizations.eyebrow')}</p>
        <h1>{t('organizations.title')}</h1>
        <p>{t('organizations.description')}</p>
      </header>

      <section className="panel" aria-labelledby="organizations-title">
        <div className="panel-header flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <Building2 className="size-4" />
            </span>
            <div>
              <h2 id="organizations-title" className="panel-title">
                {t('organizations.listTitle')}
              </h2>
              <p className="panel-description">
                {meta ? t('organizations.total', { count: meta.total }) : t('common.loading')}
              </p>
            </div>
          </div>
          <form role="search" onSubmit={submitSearch} className="flex gap-2">
            <Input
              aria-label={t('organizations.search')}
              placeholder={t('organizations.searchPlaceholder')}
              maxLength={100}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="w-full sm:w-56"
            />
            <Button
              type="submit"
              size="icon"
              variant="outline"
              aria-label={t('organizations.search')}
            >
              <Search aria-hidden="true" />
            </Button>
          </form>
        </div>

        <div className="panel-body overflow-x-auto p-0">
          {organizations.isPending ? (
            <div className="flex flex-col gap-2 p-4" aria-busy="true">
              {Array.from({ length: 4 }, (_, index) => (
                <Skeleton key={index} className="h-9 w-full" />
              ))}
            </div>
          ) : organizations.isError ? (
            <p role="alert" className="p-4 text-sm text-destructive">
              {t('organizations.errors.unavailable')}
            </p>
          ) : organizations.data.data.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">{t('organizations.empty')}</p>
          ) : (
            <table className="w-full min-w-[560px] text-sm">
              <thead className="border-b text-left text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('organizations.columns.organization')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('organizations.columns.members')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('organizations.columns.created')}
                  </th>
                  <th scope="col" className="px-4 py-2 text-right font-medium">
                    {t('organizations.columns.actions')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {organizations.data.data.map((organization) => (
                  <tr key={organization.id} className="border-b last:border-0">
                    <td className="px-4 py-2">
                      <div className="font-medium">{organization.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {organization.slug} · #{organization.id}
                      </div>
                    </td>
                    <td className="px-4 py-2 tabular-nums">{organization.member_count}</td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {dateFormat.format(new Date(organization.created_at))}
                    </td>
                    <td className="px-4 py-2 text-right">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        aria-label={t('organizations.openNamed', { name: organization.name })}
                        onClick={() => onOpen(organization.id)}
                      >
                        {t('organizations.open')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <Pager
          meta={meta}
          label={t('organizations.pagination')}
          onPageChange={(page) => onSearchChange({ ...search, page })}
        />
      </section>
    </div>
  );
}
