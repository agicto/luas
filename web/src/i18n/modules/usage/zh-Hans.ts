const messages = {
  title: '用量',
  description: '查看当前 UTC 周期内的业务用量和有效限额。',
  user: {
    title: '个人用量',
    description: '当前账户的有限指标目录。',
  },
  organization: {
    title: '组织用量',
    description: '当前组织的有限指标目录，仅组织管理员可见。',
  },
  columns: {
    metric: '指标',
    used: '已用',
    limit: '限额',
    remaining: '剩余',
    period: 'UTC 周期',
  },
  metrics: {
    apiRequests: 'API 请求',
    aiInputTokens: 'AI 输入 Token',
    aiOutputTokens: 'AI 输出 Token',
    assetTransferBytes: '资产传输字节',
    workflowRuns: '工作流运行',
  },
  units: {
    request: '次请求',
    token: 'Token',
    byte: '字节',
    run: '次运行',
  },
  unlimited: '不限额',
  notApplicable: '不适用',
  overage: '超出 {count}',
  retry: '重试',
  errors: {
    forbidden: '你没有查看此用量的权限。',
    generic: '暂时无法加载用量。',
    invalidResponse: '用量服务返回了无效数据。',
    unavailable: '用量服务暂时不可用。',
  },
} as const;

export default messages;
export type UsageMessages = typeof messages;
