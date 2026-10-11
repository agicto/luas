'use client';

import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useLocale } from 'next-intl';
import {
  AlertCircle,
  BookOpen,
  FileText,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import {
  useCreateKnowledgeArticle,
  useDeleteKnowledgeArticle,
  useKnowledgeArticle,
  useKnowledgeArticles,
  useKnowledgeSearch,
  useUpdateKnowledgeArticle,
} from '@/features/knowledge/hooks/use-knowledge';
import {
  KNOWLEDGE_QUERY_MAX_LENGTH,
  KNOWLEDGE_TITLE_MAX_LENGTH,
  createKnowledgeArticleSchema,
  updateKnowledgeArticleSchema,
} from '@/features/knowledge/schemas';
import type { KnowledgeArticle } from '@/features/knowledge/types';
import { resolveKnowledgeErrorKey } from '@/features/knowledge/utils/knowledge-error';
import {
  useOrganizationContext,
  useOrganizations,
} from '@/features/organization/hooks/use-organizations';
import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import { useT } from '@/i18n';

const SEARCH_DELAY_MS = 300;
const positiveId = /^[1-9]\d{0,15}$/u;

/**
 * Knowledge console: members browse and search the active organization's articles; owners and
 * admins also create, edit, and delete them. The organization and the open article live in the
 * URL (`?organization=&article=`) so a view can be shared and survives a reload. The API remains
 * the authorization boundary; hiding write controls is only UX.
 */
export function KnowledgeWorkspace() {
  const t = useT('knowledge');
  const organizations = useOrganizations();
  const searchParams = useSearchParams();
  const requestedOrganization = parseId(searchParams.get('organization'));
  const items = organizations.data?.items ?? [];
  const organizationId =
    items.find(item => item.id === requestedOrganization)?.id ?? items[0]?.id ?? null;
  const navigate = useKnowledgeNavigation();

  if (organizations.isPending) return <WorkspaceSkeleton />;
  if (organizations.error) {
    return (
      <ErrorAlert
        message={t(resolveKnowledgeErrorKey(organizations.error))}
        retryLabel={t('actions.retry')}
        onRetry={() => void organizations.refetch()}
      />
    );
  }
  if (organizationId === null) {
    return (
      <EmptyState
        icon={<BookOpen className="size-5" aria-hidden="true" />}
        title={t('empty.noOrganizationTitle')}
        description={t('empty.noOrganization')}
      />
    );
  }

  return (
    <OrganizationKnowledge
      key={organizationId}
      organizationId={organizationId}
      organizations={items.map(({ id, name }) => ({ id, name }))}
      selectedArticleId={parseId(searchParams.get('article'))}
      onOrganizationChange={id => navigate({ organization: id, article: null })}
      onArticleChange={id => navigate({ organization: organizationId, article: id })}
    />
  );
}

function OrganizationKnowledge({
  organizationId,
  organizations,
  selectedArticleId,
  onOrganizationChange,
  onArticleChange,
}: {
  organizationId: number;
  organizations: { id: number; name: string }[];
  selectedArticleId: number | null;
  onOrganizationChange: (organizationId: number) => void;
  onArticleChange: (articleId: number | null) => void;
}) {
  const t = useT('knowledge');
  const context = useOrganizationContext(organizationId);
  const canManage = context.data?.role === 'owner' || context.data?.role === 'admin';
  const [searchInput, setSearchInput] = useState('');
  const query = useDebouncedValue(searchInput.trim(), SEARCH_DELAY_MS);
  const [editor, setEditor] = useState<KnowledgeArticle | 'create' | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<KnowledgeArticle | null>(null);

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="w-full space-y-2 sm:max-w-xs">
          <Label htmlFor="knowledge-organization">{t('fields.organization')}</Label>
          <Select
            value={String(organizationId)}
            onValueChange={value => onOrganizationChange(Number(value))}
          >
            <SelectTrigger id="knowledge-organization" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {organizations.map(organization => (
                <SelectItem key={organization.id} value={String(organization.id)}>
                  {organization.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {canManage ? (
          <Button
            type="button"
            icon={<Plus className="size-4" />}
            onClick={() => setEditor('create')}
          >
            {t('actions.create')}
          </Button>
        ) : null}
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,20rem)_minmax(0,1fr)]">
        <section aria-labelledby="knowledge-articles-heading" className="min-w-0 space-y-3">
          <h2 id="knowledge-articles-heading" className="sr-only">
            {t('list.title')}
          </h2>
          <div className="relative">
            <Search
              className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden="true"
            />
            <Input
              type="search"
              value={searchInput}
              maxLength={KNOWLEDGE_QUERY_MAX_LENGTH}
              aria-label={t('search.label')}
              placeholder={t('search.placeholder')}
              className="pl-9"
              onChange={event => setSearchInput(event.target.value)}
            />
          </div>
          {query === '' ? (
            <ArticleList
              organizationId={organizationId}
              selectedArticleId={selectedArticleId}
              onSelect={onArticleChange}
            />
          ) : (
            <SearchResults
              organizationId={organizationId}
              query={query}
              selectedArticleId={selectedArticleId}
              onSelect={onArticleChange}
            />
          )}
        </section>

        <section aria-live="polite" className="min-w-0">
          {selectedArticleId === null ? (
            <EmptyState
              icon={<FileText className="size-5" aria-hidden="true" />}
              title={t('empty.noSelectionTitle')}
              description={t('empty.noSelection')}
            />
          ) : (
            <ArticleView
              organizationId={organizationId}
              articleId={selectedArticleId}
              canManage={canManage}
              onEdit={article => setEditor(article)}
              onDelete={article => setDeleteTarget(article)}
            />
          )}
        </section>
      </div>

      {editor !== null ? (
        <ArticleEditor
          organizationId={organizationId}
          article={editor === 'create' ? null : editor}
          onClose={() => setEditor(null)}
          onCreated={article => onArticleChange(article.id)}
        />
      ) : null}
      <DeleteArticleDialog
        organizationId={organizationId}
        article={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onDeleted={() => onArticleChange(null)}
      />
    </div>
  );
}

function ArticleList({
  organizationId,
  selectedArticleId,
  onSelect,
}: {
  organizationId: number;
  selectedArticleId: number | null;
  onSelect: (articleId: number) => void;
}) {
  const t = useT('knowledge');
  const dateTime = useDateTimeFormat();
  const articles = useKnowledgeArticles(organizationId);

  if (articles.isPending) return <ListSkeleton />;
  if (articles.error && !articles.data) {
    return (
      <ErrorAlert
        message={t(resolveKnowledgeErrorKey(articles.error))}
        retryLabel={t('actions.retry')}
        onRetry={() => void articles.refetch()}
      />
    );
  }
  const items = articles.data?.pages.flatMap(page => page.items) ?? [];
  if (items.length === 0) {
    return <p className="py-6 text-sm text-muted-foreground">{t('empty.noArticles')}</p>;
  }

  return (
    <div className="space-y-3">
      <ul className="divide-y rounded-md border">
        {items.map(article => (
          <li key={article.id}>
            <ArticleButton
              selected={article.id === selectedArticleId}
              title={article.title}
              meta={t('list.updated', { time: dateTime.format(new Date(article.updated_at)) })}
              onClick={() => onSelect(article.id)}
            />
          </li>
        ))}
      </ul>
      {articles.error ? (
        <p role="alert" className="text-sm text-error">
          {t(resolveKnowledgeErrorKey(articles.error))}
        </p>
      ) : null}
      {articles.hasNextPage ? (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          loading={articles.isFetchingNextPage}
          onClick={() => void articles.fetchNextPage()}
        >
          {t('actions.loadMore')}
        </Button>
      ) : null}
    </div>
  );
}

function SearchResults({
  organizationId,
  query,
  selectedArticleId,
  onSelect,
}: {
  organizationId: number;
  query: string;
  selectedArticleId: number | null;
  onSelect: (articleId: number) => void;
}) {
  const t = useT('knowledge');
  const search = useKnowledgeSearch(organizationId, query);

  if (search.isPending) return <ListSkeleton />;
  if (search.error) {
    return (
      <ErrorAlert
        message={t(resolveKnowledgeErrorKey(search.error))}
        retryLabel={t('actions.retry')}
        onRetry={() => void search.refetch()}
      />
    );
  }
  const hits = search.data ?? [];
  if (hits.length === 0) {
    return <p className="py-6 text-sm text-muted-foreground">{t('search.noResults')}</p>;
  }

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground" aria-live="polite">
        {t('search.results', { count: hits.length })}
      </p>
      <ul className="divide-y rounded-md border">
        {hits.map(hit => (
          <li key={hit.id}>
            <ArticleButton
              selected={hit.id === selectedArticleId}
              title={hit.title}
              meta={hit.excerpt}
              onClick={() => onSelect(hit.id)}
            />
          </li>
        ))}
      </ul>
    </div>
  );
}

function ArticleButton({
  selected,
  title,
  meta,
  onClick,
}: {
  selected: boolean;
  title: string;
  meta: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-current={selected ? 'true' : undefined}
      className={
        'block w-full px-3 py-2.5 text-left transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset ' +
        (selected ? 'bg-muted' : '')
      }
      onClick={onClick}
    >
      <span className="block truncate text-sm font-medium">{title}</span>
      {meta ? (
        <span className="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">{meta}</span>
      ) : null}
    </button>
  );
}

function ArticleView({
  organizationId,
  articleId,
  canManage,
  onEdit,
  onDelete,
}: {
  organizationId: number;
  articleId: number;
  canManage: boolean;
  onEdit: (article: KnowledgeArticle) => void;
  onDelete: (article: KnowledgeArticle) => void;
}) {
  const t = useT('knowledge');
  const dateTime = useDateTimeFormat();
  const article = useKnowledgeArticle(organizationId, articleId);

  if (article.isPending) {
    return (
      <div className="space-y-3" aria-hidden="true">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }
  if (article.error || !article.data) {
    return (
      <ErrorAlert
        message={t(resolveKnowledgeErrorKey(article.error))}
        retryLabel={t('actions.retry')}
        onRetry={() => void article.refetch()}
      />
    );
  }
  const current = article.data;

  return (
    <article className="space-y-4" aria-labelledby="knowledge-article-title">
      <header className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h2 id="knowledge-article-title" className="break-words text-xl font-semibold">
            {current.title}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('article.updated', { time: dateTime.format(new Date(current.updated_at)) })}
          </p>
        </div>
        {canManage ? (
          <div className="flex shrink-0 gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              icon={<Pencil className="size-4" />}
              onClick={() => onEdit(current)}
            >
              {t('actions.edit')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              icon={<Trash2 className="size-4" />}
              onClick={() => onDelete(current)}
            >
              {t('actions.delete')}
            </Button>
          </div>
        ) : null}
      </header>
      {current.body === '' ? (
        <p className="text-sm text-muted-foreground">{t('article.emptyBody')}</p>
      ) : (
        // Markdown is shown as source text: no HTML is rendered from article content.
        <div className="whitespace-pre-wrap break-words text-sm leading-6">{current.body}</div>
      )}
    </article>
  );
}

function ArticleEditor({
  organizationId,
  article,
  onClose,
  onCreated,
}: {
  organizationId: number;
  article: KnowledgeArticle | null;
  onClose: () => void;
  onCreated: (article: KnowledgeArticle) => void;
}) {
  const t = useT('knowledge');
  const [title, setTitle] = useState(article?.title ?? '');
  const [body, setBody] = useState(article?.body ?? '');
  const [formError, setFormError] = useState<string>();
  const create = useCreateKnowledgeArticle(organizationId);
  const update = useUpdateKnowledgeArticle(organizationId);
  const mutationError = create.error ?? update.error;
  const pending = create.isPending || update.isPending;
  const conflict =
    mutationError instanceof ApiError &&
    mutationError.errorCode === ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT;

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(undefined);
    try {
      if (article) {
        const changes = {
          ...(title.trim() === article.title ? {} : { title: title.trim() }),
          ...(body === article.body ? {} : { body }),
        };
        if (Object.keys(changes).length === 0) {
          onClose();
          return;
        }
        const input = updateKnowledgeArticleSchema.safeParse(changes);
        if (!input.success) {
          setFormError(t('errors.invalidArticle'));
          return;
        }
        await update.mutateAsync({
          articleId: article.id,
          input: input.data,
          expectedVersion: article.version,
        });
        toast.success(t('messages.updated'));
      } else {
        const input = createKnowledgeArticleSchema.safeParse({ title: title.trim(), body });
        if (!input.success) {
          setFormError(t('errors.invalidArticle'));
          return;
        }
        const created = await create.mutateAsync(input.data);
        toast.success(t('messages.created'));
        onCreated(created);
      }
      onClose();
    } catch {
      // The localized mutation error stays visible in the dialog.
    }
  };

  return (
    <Dialog open onOpenChange={open => !open && !pending && onClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t(article ? 'editor.editTitle' : 'editor.createTitle')}</DialogTitle>
          <DialogDescription>{t('editor.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} aria-busy={pending}>
          <DialogBody className="space-y-5">
            {formError || mutationError ? (
              <Alert variant="destructive">
                <AlertCircle className="size-4" aria-hidden="true" />
                <AlertDescription>
                  {formError ?? t(resolveKnowledgeErrorKey(mutationError))}
                  {conflict ? ` ${t('editor.conflictHint')}` : null}
                </AlertDescription>
              </Alert>
            ) : null}
            <div className="space-y-2">
              <Label htmlFor="knowledge-article-title-input">{t('fields.title')}</Label>
              <Input
                id="knowledge-article-title-input"
                value={title}
                maxLength={KNOWLEDGE_TITLE_MAX_LENGTH}
                autoComplete="off"
                required
                disabled={pending}
                placeholder={t('fields.titlePlaceholder')}
                onChange={event => setTitle(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="knowledge-article-body-input">{t('fields.body')}</Label>
              <Textarea
                id="knowledge-article-body-input"
                value={body}
                rows={14}
                disabled={pending}
                placeholder={t('fields.bodyPlaceholder')}
                className="font-mono text-sm"
                onChange={event => setBody(event.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t('fields.bodyHint')}</p>
            </div>
          </DialogBody>
          <DialogFooter className="mt-4">
            <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
              {t('actions.cancel')}
            </Button>
            <Button type="submit" loading={pending}>
              {t(article ? 'actions.save' : 'actions.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function DeleteArticleDialog({
  organizationId,
  article,
  onClose,
  onDeleted,
}: {
  organizationId: number;
  article: KnowledgeArticle | null;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const t = useT('knowledge');
  const remove = useDeleteKnowledgeArticle(organizationId);

  const confirm = async () => {
    if (!article) return;
    try {
      await remove.mutateAsync({ articleId: article.id, expectedVersion: article.version });
      toast.success(t('messages.deleted'));
      onDeleted();
      onClose();
    } catch (error) {
      toast.error(t(resolveKnowledgeErrorKey(error)));
    }
  };

  return (
    <Dialog open={article !== null} onOpenChange={open => !open && !remove.isPending && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t('delete.title')}</DialogTitle>
          <DialogDescription>
            {t('delete.description', { title: article?.title ?? '' })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" disabled={remove.isPending} onClick={onClose}>
            {t('actions.cancel')}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={remove.isPending}
            icon={<Trash2 className="size-4" />}
            onClick={() => void confirm()}
          >
            {t('actions.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ErrorAlert({
  message,
  retryLabel,
  onRetry,
}: {
  message: string;
  retryLabel: string;
  onRetry: () => void;
}) {
  return (
    <Alert variant="destructive">
      <AlertCircle className="size-4" aria-hidden="true" />
      <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
        <span>{message}</span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          icon={<RefreshCw className="size-3.5" />}
          onClick={onRetry}
        >
          {retryLabel}
        </Button>
      </AlertDescription>
    </Alert>
  );
}

function EmptyState({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-center rounded-md border border-dashed px-6 py-12 text-center">
      <div className="flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {icon}
      </div>
      <h2 className="mt-3 text-sm font-medium">{title}</h2>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

function ListSkeleton() {
  return (
    <div className="space-y-2" aria-hidden="true">
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
    </div>
  );
}

function WorkspaceSkeleton() {
  return (
    <div className="space-y-5" aria-hidden="true">
      <Skeleton className="h-10 w-full max-w-xs" />
      <div className="grid gap-6 lg:grid-cols-[20rem_1fr]">
        <ListSkeleton />
        <Skeleton className="h-48 w-full" />
      </div>
    </div>
  );
}

function useKnowledgeNavigation() {
  const router = useRouter();
  const pathname = usePathname();
  return ({ organization, article }: { organization: number; article: number | null }) => {
    const params = new URLSearchParams({ organization: String(organization) });
    if (article !== null) params.set('article', String(article));
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };
}

function useDebouncedValue(value: string, delay: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

function useDateTimeFormat(): Intl.DateTimeFormat {
  const locale = useLocale();
  return useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }),
    [locale]
  );
}

function parseId(value: string | null): number | null {
  if (value === null || !positiveId.test(value)) return null;
  const id = Number(value);
  return Number.isSafeInteger(id) ? id : null;
}
