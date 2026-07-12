const messages = {
  nav: {
    home: '首页',
    console: '控制台',
  },
  footer: {
    tagline: 'AI 时代全栈脚手架',
    copyright: '© {year} Luas。保留所有权利。',
  },
  hero: {
    eyebrow: '面向 AI 协作的企业脚手架',
    titlePrefix: '快速构建现代应用',
    titleAccent: '从清晰架构开始',
    description:
      'Luas 将 Next.js Web 与 Go API 作为可独立部署、通过契约协作的两个部分，内置认证、国际化与工程治理。',
    primaryAction: '开始使用',
    secondaryAction: '查看控制台',
  },
  features: {
    title: '业务起步所需的基础能力',
    description: '保留清晰边界的同时，提供可替换、可验证的企业级起点。',
    authentication: {
      title: '认证起步模块',
      description: '提供登录、注册、会话与受保护路由的完整示例。',
    },
    console: {
      title: '可替换控制台',
      description: '提供已认证工作区、设置入口与开发工具边界。',
    },
    i18n: {
      title: '类型安全国际化',
      description: '提供按路由加载、键与作用域均可推导的翻译体系。',
    },
    ui: {
      title: '语义化界面基础',
      description: '通过设计令牌和可访问组件支持持续扩展。',
    },
  },
  techStack: {
    title: '现代技术栈',
    description: '采用成熟工具，兼顾运行性能、类型安全与开发体验。',
  },
  cta: {
    title: '准备开始构建？',
    description: '从 Luas 的契约、认证和可替换控制台开始你的下一个项目。',
    primaryAction: '免费开始',
    githubAction: '在 GitHub 查看',
  },
};

export default messages;
export type SiteMessages = typeof messages;
