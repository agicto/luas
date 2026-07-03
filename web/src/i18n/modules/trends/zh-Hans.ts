const messages = {
  title: '热点流水线',
  description: 'daily.dev 热点同步、AI 初筛和文章选题入口',
  syncNow: '立即同步',
  refreshing: '同步中',
  searchPlaceholder: '搜索标题、摘要或频道',
  source: '热点源',
  lastPolled: '最近同步',
  interval: '{count} 分钟一次',
  noSource: '尚未建立数据源',
  tabs: {
    all: '全部',
    candidate: '候选',
    new: '新热点',
    selected: '已选择',
    rejected: '已拒绝',
  },
  stats: {
    total: '总热点',
    candidate: '候选热点',
    new: '待观察',
    queued: '评分队列',
  },
  table: {
    topic: '热点',
    channel: '频道',
    score: '评分',
    audience: '受众',
    status: '状态',
    time: '时间',
    action: '链接',
  },
  score: {
    h: '热度',
    k: '知识',
    r: '相关',
    brand: '品牌',
    risk: '风险',
    total: '总分',
  },
  status: {
    new: '新热点',
    candidate: '候选',
    selected: '已选择',
    rejected: '已拒绝',
    archived: '已归档',
  },
  empty: {
    title: '暂无匹配热点',
    description: '可以切换筛选条件，或手动触发一次 daily.dev 同步。',
  },
  errors: {
    load: '热点数据加载失败',
    sync: '同步失败',
  },
  toast: {
    syncSuccess: '同步完成：新增 {inserted} 条，候选 {candidates} 条',
  },
};

export default messages;

export type TrendsMessages = typeof messages;
