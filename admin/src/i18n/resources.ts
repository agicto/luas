type LocaleResourceShape<T> = T extends string
  ? string
  : { [Key in keyof T]: LocaleResourceShape<T[Key]> };

const enUS = {
  translation: {
    app: {
      name: 'Luas',
      shell: 'Static Console',
    },
    navigation: {
      overview: 'Overview',
      preferences: 'Preferences',
      console: 'Console',
      section: 'Workspace',
      breadcrumb: 'Breadcrumb',
      toggle: 'Toggle sidebar',
      mobileDescription: 'Navigate the static console.',
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
      shell: '静态控制台',
    },
    navigation: {
      overview: '概览',
      preferences: '偏好设置',
      console: '控制台',
      section: '工作区',
      breadcrumb: '面包屑导航',
      toggle: '切换侧栏',
      mobileDescription: '浏览静态控制台功能。',
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
