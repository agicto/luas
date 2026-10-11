import type { LocaleMessageShape } from '../../locale-message-shape';
import type { KnowledgeMessages } from './en-US';

const messages = {
  title: '知识库',
  description: '阅读和搜索组织保存的文章。组织所有者和管理员可以编辑文章。',
  fields: {
    organization: '组织',
    title: '标题',
    titlePlaceholder: '退款处理流程',
    body: '内容',
    bodyPlaceholder: '使用 Markdown 编写。',
    bodyHint: 'Markdown 按原文保存，并以纯文本显示。',
  },
  list: {
    title: '文章',
    updated: '更新于 {time}',
  },
  search: {
    label: '搜索文章',
    placeholder: '搜索标题和内容',
    results: '匹配结果：{count}',
    noResults: '没有与此搜索匹配的文章。',
  },
  article: {
    updated: '最后更新于 {time}',
    emptyBody: '这篇文章还没有内容。',
  },
  editor: {
    createTitle: '新建文章',
    editTitle: '编辑文章',
    description: '组织的所有成员都可以阅读和搜索这篇文章。',
    conflictHint: '关闭此对话框以加载最新版本，然后重新进行修改。',
  },
  delete: {
    title: '删除文章',
    description: '删除“{title}”？成员将无法再找到它。此操作无法撤销。',
  },
  empty: {
    noOrganizationTitle: '暂无组织',
    noOrganization: '创建或加入一个组织后，即可保存共享的知识文章。',
    noArticles: '暂无文章。',
    noSelectionTitle: '未选择文章',
    noSelection: '从列表或搜索结果中选择一篇文章，在此阅读。',
  },
  actions: {
    create: '新建文章',
    edit: '编辑',
    delete: '删除',
    save: '保存修改',
    cancel: '取消',
    retry: '重试',
    loadMore: '加载更多',
  },
  messages: {
    created: '文章已创建',
    updated: '文章已保存',
    deleted: '文章已删除',
  },
  errors: {
    forbidden: '只有组织所有者和管理员可以修改文章。',
    generic: '暂时无法完成知识库请求。',
    invalidArticle: '标题不超过 200 个字符，内容不超过 200,000 个字符。',
    invalidResponse: '知识库服务返回了无效数据。',
    notFound: '这篇文章不存在或已被删除。',
    organizationUnavailable: '你的账户无法访问此组织。',
    preconditionRequired: '请重新加载文章后再修改。',
    tooLarge: '文章过大，无法保存。',
    unavailable: '知识库服务暂时不可用。',
    versionConflict: '自你打开这篇文章后，其他人已修改了它。',
  },
} as const satisfies LocaleMessageShape<KnowledgeMessages>;

export default messages;
