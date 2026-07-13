const messages = {
  notifications: '通知',
  profile: '个人资料',
  returnToSite: '返回站点',
  greeting: {
    hello: '你好，{name}',
    late: '夜深了，{name}',
    morning: '早上好，{name}',
    afternoon: '下午好，{name}',
    evening: '晚上好，{name}',
  },
  home: {
    defaultUser: '朋友',
    welcomeDescription: '这是可替换的 Luas 控制台起始页，请用你的业务工作区替换它。',
    openSettings: '打开设置',
    open: '打开',
    nextSteps: '后续步骤',
    quickLinks: {
      apiDocs: {
        title: 'API 文档',
        description: '查看 Luas Go 后端的 OpenAPI 规范。',
      },
      styleguide: {
        title: '设计规范',
        description: '浏览设计系统和组件陈列。',
      },
      i18nTest: {
        title: '国际化测试',
        description: '检查 next-intl 翻译树。',
      },
    },
    steps: {
      apiBefore: '将',
      apiAfter: '指向契约兼容的生产端点或适配器。',
      replaceBefore: '用真实业务工作区替换此页面（',
      replaceAfter: '）。',
      featuresBefore: '在',
      featuresBetween: '中添加新 feature，并通过',
      featuresAfter: '下的新路由公开它们。',
    },
  },
} as const;

export default messages;
export type ConsoleMessages = typeof messages;
