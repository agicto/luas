import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { KnowledgeWorkspace } from '@/features/knowledge/components/knowledge-workspace';
import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';
import { messages } from '@/i18n/modules';

const navigation = vi.hoisted(() => ({
  replace: vi.fn(),
  params: new URLSearchParams(),
}));
const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
const state = vi.hoisted(() => ({
  role: 'owner' as string,
  articles: {
    data: undefined as unknown,
    isPending: false,
    error: null as unknown,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
    refetch: vi.fn(),
  },
  article: {
    data: undefined as unknown,
    isPending: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
  search: {
    data: undefined as unknown,
    isPending: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
  searchCalls: [] as string[],
  create: { isPending: false, error: null as unknown, mutateAsync: vi.fn() },
  update: { isPending: false, error: null as unknown, mutateAsync: vi.fn() },
  remove: { isPending: false, error: null as unknown, mutateAsync: vi.fn() },
}));

vi.mock('next/navigation', () => ({
  usePathname: () => '/console/knowledge',
  useRouter: () => ({ replace: navigation.replace }),
  useSearchParams: () => navigation.params,
}));
vi.mock('sonner', () => ({ toast }));
vi.mock('@/features/organization/hooks/use-organizations', () => ({
  useOrganizations: () => ({
    data: {
      items: [
        { id: 1, name: 'Acme' },
        { id: 2, name: 'Globex' },
      ],
    },
    isPending: false,
    error: null,
    refetch: vi.fn(),
  }),
  useOrganizationContext: () => ({ data: { role: state.role } }),
}));
vi.mock('@/features/knowledge/hooks/use-knowledge', () => ({
  useKnowledgeArticles: () => state.articles,
  useKnowledgeArticle: () => state.article,
  useKnowledgeSearch: (_organizationId: number, query: string) => {
    state.searchCalls.push(query);
    return state.search;
  },
  useCreateKnowledgeArticle: () => state.create,
  useUpdateKnowledgeArticle: () => state.update,
  useDeleteKnowledgeArticle: () => state.remove,
}));

describe('knowledge workspace', () => {
  beforeEach(() => {
    navigation.replace.mockReset();
    navigation.params = new URLSearchParams();
    toast.success.mockReset();
    toast.error.mockReset();
    state.role = 'owner';
    state.articles.data = {
      pages: [{ items: [summary(2, 'Onboarding'), summary(1, 'Refund policy')], meta: {} }],
    };
    state.articles.error = null;
    state.articles.hasNextPage = false;
    state.articles.fetchNextPage.mockReset();
    state.article.data = article();
    state.article.error = null;
    state.search.data = [];
    state.search.error = null;
    state.searchCalls = [];
    for (const mutation of [state.create, state.update, state.remove]) {
      mutation.isPending = false;
      mutation.error = null;
      mutation.mutateAsync.mockReset();
    }
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('lists articles, pages with the cursor, and opens one through the URL', () => {
    state.articles.hasNextPage = true;
    renderWithMessages(<KnowledgeWorkspace />);

    expect(screen.getByText('Onboarding')).toBeInTheDocument();
    expect(screen.getByText('No article selected')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Load more' }));
    expect(state.articles.fetchNextPage).toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: /Refund policy/u }));
    expect(navigation.replace).toHaveBeenCalledWith('/console/knowledge?organization=1&article=1', {
      scroll: false,
    });
  });

  it('shows the selected article as text and hides write controls from members', () => {
    state.role = 'member';
    navigation.params = new URLSearchParams('organization=1&article=1');
    renderWithMessages(<KnowledgeWorkspace />);

    expect(screen.getByRole('heading', { name: 'Refund policy' })).toBeInTheDocument();
    expect(screen.getByText(/<script>alert\(1\)<\/script>/u)).toBeInTheDocument();
    expect(document.querySelector('script')).toBeNull();
    expect(screen.queryByRole('button', { name: 'New article' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Edit' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Delete' })).toBeNull();
  });

  it('debounces search and shows ranked hits with excerpts', async () => {
    vi.useFakeTimers();
    state.search.data = [
      { id: 1, title: 'Refund policy', excerpt: '七天内申请', version: 1, updated_at: now() },
    ];
    renderWithMessages(<KnowledgeWorkspace />);

    fireEvent.change(screen.getByRole('searchbox', { name: 'Search articles' }), {
      target: { value: ' 七天 ' },
    });
    expect(state.searchCalls).not.toContain('七天');
    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    expect(state.searchCalls).toContain('七天');
    expect(screen.getByText('七天内申请')).toBeInTheDocument();
    expect(screen.getByText('Matches: 1')).toBeInTheDocument();
  });

  it('edits with the version that was read and reports a conflict', async () => {
    navigation.params = new URLSearchParams('organization=1&article=1');
    state.update.mutateAsync.mockRejectedValue(
      new ApiError('conflict', ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT)
    );
    const view = renderWithMessages(<KnowledgeWorkspace />);

    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Refunds v2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));

    await waitFor(() =>
      expect(state.update.mutateAsync).toHaveBeenCalledWith({
        articleId: 1,
        input: { title: 'Refunds v2' },
        expectedVersion: 4,
      })
    );
    state.update.error = new ApiError('conflict', ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT);
    view.rerender(wrap(<KnowledgeWorkspace />));
    expect(
      await screen.findByText(/Someone else changed this article since you opened it/u)
    ).toBeInTheDocument();
  });

  it('creates an article and opens it', async () => {
    state.create.mutateAsync.mockResolvedValue({ ...article(), id: 9, title: 'New' });
    renderWithMessages(<KnowledgeWorkspace />);

    fireEvent.click(screen.getByRole('button', { name: 'New article' }));
    fireEvent.change(screen.getByLabelText('Title'), { target: { value: '  New  ' } });
    fireEvent.change(screen.getByLabelText('Content'), { target: { value: 'Body' } });
    fireEvent.click(
      within(screen.getByRole('dialog')).getByRole('button', { name: 'New article' })
    );

    await waitFor(() =>
      expect(state.create.mutateAsync).toHaveBeenCalledWith({ title: 'New', body: 'Body' })
    );
    expect(toast.success).toHaveBeenCalledWith('Article created');
    expect(navigation.replace).toHaveBeenCalledWith('/console/knowledge?organization=1&article=9', {
      scroll: false,
    });
  });

  it('deletes with the current version after confirmation', async () => {
    navigation.params = new URLSearchParams('organization=1&article=1');
    state.remove.mutateAsync.mockResolvedValue(undefined);
    renderWithMessages(<KnowledgeWorkspace />);

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));

    await waitFor(() =>
      expect(state.remove.mutateAsync).toHaveBeenCalledWith({ articleId: 1, expectedVersion: 4 })
    );
    expect(navigation.replace).toHaveBeenCalledWith('/console/knowledge?organization=1', {
      scroll: false,
    });
  });
});

function summary(id: number, title: string) {
  return { id, title, version: 1, updated_at: now() };
}

function article() {
  return {
    id: 1,
    title: 'Refund policy',
    body: '# Refunds\n\n<script>alert(1)</script>',
    version: 4,
    created_by: 1,
    updated_by: 1,
    created_at: now(),
    updated_at: now(),
  };
}

function now() {
  return '2026-10-11T08:00:00Z';
}

function wrap(children: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}

function renderWithMessages(children: React.ReactNode) {
  return render(wrap(children));
}
