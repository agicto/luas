const messages = {
  title: 'Knowledge',
  description:
    'Read and search the articles your organization keeps. Owners and admins can edit them.',
  fields: {
    organization: 'Organization',
    title: 'Title',
    titlePlaceholder: 'How we handle refunds',
    body: 'Content',
    bodyPlaceholder: 'Write in Markdown.',
    bodyHint: 'Markdown is stored as written and shown as plain text.',
  },
  list: {
    title: 'Articles',
    updated: 'Updated {time}',
  },
  search: {
    label: 'Search articles',
    placeholder: 'Search titles and content',
    results: 'Matches: {count}',
    noResults: 'No articles match this search.',
  },
  article: {
    updated: 'Last updated {time}',
    emptyBody: 'This article has no content yet.',
  },
  editor: {
    createTitle: 'New article',
    editTitle: 'Edit article',
    description: 'Every member of the organization can read and search this article.',
    conflictHint: 'Close this dialog to load the latest version, then apply your changes again.',
  },
  delete: {
    title: 'Delete article',
    description: 'Delete "{title}"? Members will no longer find it. This cannot be undone.',
  },
  empty: {
    noOrganizationTitle: 'No organization yet',
    noOrganization: 'Create or join an organization to keep shared knowledge articles.',
    noArticles: 'No articles yet.',
    noSelectionTitle: 'No article selected',
    noSelection: 'Choose an article from the list or search to read it here.',
  },
  actions: {
    create: 'New article',
    edit: 'Edit',
    delete: 'Delete',
    save: 'Save changes',
    cancel: 'Cancel',
    retry: 'Retry',
    loadMore: 'Load more',
  },
  messages: {
    created: 'Article created',
    updated: 'Article saved',
    deleted: 'Article deleted',
  },
  errors: {
    forbidden: 'Only organization owners and admins can change articles.',
    generic: 'The knowledge request could not be completed.',
    invalidArticle:
      'Enter a title of up to 200 characters and content of up to 200,000 characters.',
    invalidResponse: 'The knowledge service returned invalid data.',
    notFound: 'This article does not exist or was deleted.',
    organizationUnavailable: 'This organization is not available to your account.',
    preconditionRequired: 'Reload the article before changing it.',
    tooLarge: 'The article is too large to save.',
    unavailable: 'The knowledge service is temporarily unavailable.',
    versionConflict: 'Someone else changed this article since you opened it.',
  },
} as const;

export default messages;
export type KnowledgeMessages = typeof messages;
