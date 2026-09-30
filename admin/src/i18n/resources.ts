type LocaleResourceShape<T> = T extends string
  ? string
  : { [Key in keyof T]: LocaleResourceShape<T[Key]> };

const enUS = {
  translation: {
    app: {
      name: 'Luas',
      shell: 'Admin Console',
    },
    navigation: {
      overview: 'Overview',
      preferences: 'Preferences',
      users: 'Users',
      console: 'Console',
      section: 'Workspace',
      breadcrumb: 'Breadcrumb',
      toggle: 'Toggle sidebar',
      mobileDescription: 'Navigate the project management console.',
      open: 'Open navigation',
      close: 'Close navigation',
    },
    overview: {
      eyebrow: 'Workspace',
      title: 'System overview',
      description: 'Current browser runtime and API availability.',
      api: 'API',
      apiAvailable: 'Available',
      apiUnavailable: 'Unavailable',
      apiChecking: 'Checking',
      delivery: 'Delivery',
      deliveryValue: 'OSS / CDN',
      runtime: 'Runtime',
      runtimeValue: 'Static browser',
      readinessTitle: 'API readiness',
      readinessDescription: 'Live result from the configured Go API health endpoint.',
      refresh: 'Refresh API status',
      checked: 'Checked {{time}}',
      waiting: 'Waiting for a readiness response.',
      unreachable: 'The API could not be reached from this browser.',
      statusValue: 'API status: {{status}}',
      statusUp: 'Up',
      statusDegraded: 'Degraded',
      requestId: 'Request ID',
      releaseTitle: 'Release profile',
      releaseDescription: 'Properties of the current static build.',
      buildMode: 'Build mode',
      basePath: 'Base path',
      apiBase: 'API base',
      featureCount: 'Core features',
    },
    preferences: {
      eyebrow: 'Workspace',
      title: 'Preferences',
      description: 'Browser-local display settings for this console.',
      appearanceTitle: 'Appearance',
      appearanceDescription: 'Choose how the console follows your display.',
      theme: 'Theme',
      light: 'Light',
      dark: 'Dark',
      system: 'System',
      languageTitle: 'Language',
      languageDescription: 'Select the interface language stored in this browser.',
      language: 'Language',
      english: 'English',
      chinese: '简体中文',
      storageNote: 'Display preferences contain no account or credential data.',
    },
    users: {
      eyebrow: 'Operations',
      title: 'Users',
      description: 'Find accounts, disable or re-enable them, and end their sessions.',
      listTitle: 'Accounts',
      total_one: '{{count}} account',
      total_other: '{{count}} accounts',
      loading: 'Loading accounts…',
      search: 'Search accounts',
      searchPlaceholder: 'Username or email',
      statusFilter: 'Filter by status',
      status: { all: 'All', active: 'Active', disabled: 'Disabled' },
      operator: 'Operator',
      never: 'Never',
      empty: 'No accounts match these filters.',
      protectedHint: 'Managed from the CLI',
      columns: {
        account: 'Account',
        status: 'Status',
        lastLogin: 'Last sign-in',
        actions: 'Actions',
      },
      actions: { disable: 'Disable', enable: 'Enable', revokeSessions: 'End sessions' },
      confirm: {
        disable: {
          title: 'Disable this account?',
          description:
            '{{name}} will be signed out everywhere and cannot sign in until re-enabled.',
        },
        enable: {
          title: 'Enable this account?',
          description: '{{name}} will be able to sign in again. Ended sessions are not restored.',
        },
        revokeSessions: {
          title: 'End all sessions?',
          description: '{{name}} will be signed out everywhere but can sign in again.',
        },
      },
      cancel: 'Cancel',
      pagination: 'Pagination',
      page: 'Page {{page}} of {{pages}}',
      previous: 'Previous',
      next: 'Next',
      errors: {
        protected: 'Operator accounts can only be changed from the server command line.',
        notFound: 'This account no longer exists.',
        unavailable: 'Accounts are unavailable right now. Try again shortly.',
      },
    },
    auth: {
      title: 'Operator sign-in',
      description: 'Sign in with an account that has platform-operator access.',
      identifier: 'Username or email',
      password: 'Password',
      submit: 'Sign in',
      submitting: 'Signing in…',
      signOut: 'Sign out',
      operatorNote:
        'Operator access is granted from the server command line, not from this console.',
      errors: {
        invalidCredentials: 'The username or password is incorrect.',
        notOperator: 'This account does not have platform-operator access.',
        rateLimited: 'Too many sign-in attempts. Wait a moment and try again.',
        originRejected:
          'This console address is not allowed to sign in. Check OPERATOR_ALLOWED_ORIGINS.',
        unavailable: 'Sign-in is unavailable right now. Try again shortly.',
      },
    },
    errors: {
      notFoundTitle: 'Page not found',
      notFoundDescription: 'The requested console route does not exist.',
      backToConsole: 'Back to console',
    },
    common: {
      adminConsole: 'Admin Console',
      development: 'Development',
      production: 'Production',
    },
  },
} as const;

const zhHans = {
  translation: {
    app: {
      name: 'Luas',
      shell: '管理后台',
    },
    navigation: {
      overview: '概览',
      preferences: '偏好设置',
      users: '用户',
      console: '控制台',
      section: '工作区',
      breadcrumb: '面包屑导航',
      toggle: '切换侧栏',
      mobileDescription: '浏览项目管理后台功能。',
      open: '打开导航',
      close: '关闭导航',
    },
    overview: {
      eyebrow: '工作区',
      title: '系统概览',
      description: '查看当前浏览器运行环境与 API 可用性。',
      api: 'API',
      apiAvailable: '可用',
      apiUnavailable: '不可用',
      apiChecking: '检查中',
      delivery: '部署方式',
      deliveryValue: 'OSS / CDN',
      runtime: '运行环境',
      runtimeValue: '纯浏览器',
      readinessTitle: 'API 就绪状态',
      readinessDescription: '来自当前 Go API 健康检查端点的实时结果。',
      refresh: '刷新 API 状态',
      checked: '检查时间 {{time}}',
      waiting: '正在等待就绪检查结果。',
      unreachable: '当前浏览器无法访问 API。',
      statusValue: 'API 状态：{{status}}',
      statusUp: '正常',
      statusDegraded: '降级',
      requestId: '请求 ID',
      releaseTitle: '构建信息',
      releaseDescription: '当前静态构建的公开属性。',
      buildMode: '构建模式',
      basePath: '基础路径',
      apiBase: 'API 地址',
      featureCount: '核心功能',
    },
    preferences: {
      eyebrow: '工作区',
      title: '偏好设置',
      description: '仅保存在当前浏览器中的控制台显示设置。',
      appearanceTitle: '外观',
      appearanceDescription: '选择控制台如何适配你的显示环境。',
      theme: '主题',
      light: '浅色',
      dark: '深色',
      system: '跟随系统',
      languageTitle: '语言',
      languageDescription: '选择保存在当前浏览器中的界面语言。',
      language: '语言',
      english: 'English',
      chinese: '简体中文',
      storageNote: '显示偏好不包含账号或凭证数据。',
    },
    users: {
      eyebrow: '运维',
      title: '用户',
      description: '查找账号，禁用或重新启用，并结束其登录会话。',
      listTitle: '账号',
      total_one: '共 {{count}} 个账号',
      total_other: '共 {{count}} 个账号',
      loading: '正在加载账号…',
      search: '搜索账号',
      searchPlaceholder: '用户名或邮箱',
      statusFilter: '按状态筛选',
      status: { all: '全部', active: '正常', disabled: '已禁用' },
      operator: '运维员',
      never: '从未',
      empty: '没有符合筛选条件的账号。',
      protectedHint: '需在命令行管理',
      columns: { account: '账号', status: '状态', lastLogin: '最近登录', actions: '操作' },
      actions: { disable: '禁用', enable: '启用', revokeSessions: '强制下线' },
      confirm: {
        disable: {
          title: '禁用该账号？',
          description: '{{name}} 将在所有设备上退出登录，并在重新启用前无法登录。',
        },
        enable: {
          title: '启用该账号？',
          description: '{{name}} 将可以重新登录，已结束的会话不会恢复。',
        },
        revokeSessions: {
          title: '结束全部会话？',
          description: '{{name}} 将在所有设备上退出登录，但可以重新登录。',
        },
      },
      cancel: '取消',
      pagination: '分页',
      page: '第 {{page}} / {{pages}} 页',
      previous: '上一页',
      next: '下一页',
      errors: {
        protected: '运维员账号只能在服务器命令行中修改。',
        notFound: '该账号已不存在。',
        unavailable: '暂时无法加载账号，请稍后重试。',
      },
    },
    auth: {
      title: '运维员登录',
      description: '使用具有平台运维权限的账号登录。',
      identifier: '用户名或邮箱',
      password: '密码',
      submit: '登录',
      submitting: '正在登录…',
      signOut: '退出登录',
      operatorNote: '运维权限需在服务器命令行授予，无法在本控制台中申请。',
      errors: {
        invalidCredentials: '用户名或密码不正确。',
        notOperator: '该账号没有平台运维权限。',
        rateLimited: '登录尝试过于频繁，请稍后再试。',
        originRejected: '当前控制台地址不允许登录，请检查 OPERATOR_ALLOWED_ORIGINS。',
        unavailable: '登录服务暂时不可用，请稍后重试。',
      },
    },
    errors: {
      notFoundTitle: '页面不存在',
      notFoundDescription: '请求的控制台路由不存在。',
      backToConsole: '返回控制台',
    },
    common: {
      adminConsole: '管理后台',
      development: '开发环境',
      production: '生产环境',
    },
  },
} as const satisfies LocaleResourceShape<typeof enUS>;

export const resources = {
  'en-US': enUS,
  'zh-Hans': zhHans,
} as const;

export type SupportedLocale = keyof typeof resources;

type I18nextVariableNames<Message extends string> =
  Message extends `${string}{{${infer Name}}}${infer Rest}`
    ? Name | I18nextVariableNames<Rest>
    : never;

type SameVariables<Source extends string, Candidate extends string> = [
  I18nextVariableNames<Source>,
] extends [I18nextVariableNames<Candidate>]
  ? [I18nextVariableNames<Candidate>] extends [I18nextVariableNames<Source>]
    ? true
    : false
  : false;

type ResourceVariableParity<Source, Candidate> = Source extends string
  ? Candidate extends string
    ? SameVariables<Source, Candidate>
    : false
  : Source extends object
    ? Candidate extends { [Key in keyof Source]: unknown }
      ? false extends {
          [Key in keyof Source]: ResourceVariableParity<Source[Key], Candidate[Key]>;
        }[keyof Source]
        ? false
        : true
      : false
    : false;

type Assert<T extends true> = T;

/** Compile-time guard for translated i18next interpolation variable names. */
export type LocaleResourceVariableParityCheck = Assert<
  ResourceVariableParity<typeof enUS, typeof zhHans>
>;
