import { ArrowLeft, Building2, Users } from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pager } from '@/components/layout/pager';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  useOrganization,
  useOrganizationMembers,
} from '@/features/organizations/hooks/use-organizations';
import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';

interface OrganizationDetailPageProps {
  organizationId: number;
  onBack: () => void;
  /** Sections owned by other optional features, shown once the organization is known to exist. */
  children?: ReactNode;
}

export function OrganizationDetailPage({
  organizationId,
  onBack,
  children,
}: OrganizationDetailPageProps) {
  const { t, i18n } = useTranslation();
  const organization = useOrganization(organizationId);
  const notFound =
    organization.error instanceof ApiError &&
    organization.error.errorCode === ApiErrorCode.ORGANIZATION_NOT_FOUND;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('organizations.eyebrow')}</p>
        <h1>{organization.data?.name ?? t('organizations.detailTitle')}</h1>
        <div>
          <Button type="button" size="sm" variant="outline" onClick={onBack}>
            <ArrowLeft aria-hidden="true" />
            {t('organizations.back')}
          </Button>
        </div>
      </header>

      <section className="panel" aria-labelledby="organization-summary-title">
        <div className="panel-header">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <Building2 className="size-4" />
            </span>
            <div>
              <h2 id="organization-summary-title" className="panel-title">
                {t('organizations.summaryTitle')}
              </h2>
              <p className="panel-description">{t('organizations.summaryDescription')}</p>
            </div>
          </div>
        </div>
        <div className="panel-body">
          {organization.isPending ? (
            <Skeleton className="h-20 w-full" />
          ) : organization.isError ? (
            <p role="alert" className="text-sm text-destructive">
              {notFound
                ? t('organizations.errors.notFound')
                : t('organizations.errors.unavailable')}
            </p>
          ) : (
            <dl className="flex flex-col gap-2 text-sm">
              <div className="definition-row">
                <dt>{t('organizations.fields.id')}</dt>
                <dd>{organization.data.id}</dd>
              </div>
              <div className="definition-row">
                <dt>{t('organizations.fields.slug')}</dt>
                <dd>{organization.data.slug}</dd>
              </div>
              <div className="definition-row">
                <dt>{t('organizations.fields.members')}</dt>
                <dd>{organization.data.member_count}</dd>
              </div>
              <div className="definition-row">
                <dt>{t('organizations.fields.createdBy')}</dt>
                <dd>{t('organizations.userRef', { id: organization.data.created_by })}</dd>
              </div>
              <div className="definition-row">
                <dt>{t('organizations.fields.created')}</dt>
                <dd>{dateFormat.format(new Date(organization.data.created_at))}</dd>
              </div>
            </dl>
          )}
        </div>
      </section>

      {organization.isSuccess ? (
        <>
          <OrganizationMembers organizationId={organizationId} />
          {children}
        </>
      ) : null}
    </div>
  );
}

function OrganizationMembers({ organizationId }: { organizationId: number }) {
  const { t, i18n } = useTranslation();
  const [page, setPage] = useState(1);
  const members = useOrganizationMembers(organizationId, page);
  const meta = members.data?.meta;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, { dateStyle: 'medium' });

  return (
    <section className="panel" aria-labelledby="organization-members-title">
      <div className="panel-header">
        <div className="flex items-start gap-3">
          <span className="icon-surface" aria-hidden="true">
            <Users className="size-4" />
          </span>
          <div>
            <h2 id="organization-members-title" className="panel-title">
              {t('organizations.membersTitle')}
            </h2>
            <p className="panel-description">
              {meta ? t('organizations.memberTotal', { count: meta.total }) : t('common.loading')}
            </p>
          </div>
        </div>
      </div>
      <div className="panel-body overflow-x-auto p-0">
        {members.isPending ? (
          <div className="flex flex-col gap-2 p-4" aria-busy="true">
            {Array.from({ length: 3 }, (_, index) => (
              <Skeleton key={index} className="h-9 w-full" />
            ))}
          </div>
        ) : members.isError ? (
          <p role="alert" className="p-4 text-sm text-destructive">
            {t('organizations.errors.unavailable')}
          </p>
        ) : members.data.data.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{t('organizations.membersEmpty')}</p>
        ) : (
          <table className="w-full min-w-[520px] text-sm">
            <thead className="border-b text-left text-xs text-muted-foreground">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('organizations.memberColumns.account')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('organizations.memberColumns.role')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('organizations.memberColumns.joined')}
                </th>
              </tr>
            </thead>
            <tbody>
              {members.data.data.map((member) => (
                <tr key={member.id} className="border-b last:border-0">
                  <td className="px-4 py-2">
                    <div className="font-medium">{member.nickname || member.username}</div>
                    <div className="text-xs text-muted-foreground">
                      {member.username} · {member.email}
                    </div>
                  </td>
                  <td className="px-4 py-2">
                    <StatusBadge>{t(`organizations.roles.${member.role}`)}</StatusBadge>
                  </td>
                  <td className="px-4 py-2 text-muted-foreground">
                    {dateFormat.format(new Date(member.joined_at))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <Pager meta={meta} label={t('organizations.memberPagination')} onPageChange={setPage} />
    </section>
  );
}
