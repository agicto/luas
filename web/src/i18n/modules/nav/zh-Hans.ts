// Nav translations - Simplified Chinese
import type { LocaleMessageShape } from '../../locale-message-shape';
import type { NavMessages } from './en-US';

const messages = {
  home: '首页',
  console: '控制台',
  organizations: '组织',
  assets: '资产',
  usage: '用量',
  settings: '设置',
  profile: '个人资料',
  analytics: '数据分析',
  styleguide: '设计规范',
} as const satisfies LocaleMessageShape<NavMessages>;

export default messages;
