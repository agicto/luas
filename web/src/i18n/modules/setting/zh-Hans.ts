import type { LocaleMessageShape } from '../../locale-message-shape';
import type { SettingMessages } from './en-US';

const messages = {
  user: {
    title: '个人偏好',
    description: '当前账户使用的语言区域和时区。',
  },
  organization: {
    title: '组织偏好',
    description: '当前组织使用的默认语言区域。',
  },
  fields: {
    locale: '语言区域',
    localeDescription: '界面语言与区域格式。',
    timezone: '时区',
    timezoneDescription: '日期和计划使用的 IANA 时区。',
  },
  options: {
    enUS: '英语（美国）',
    zhHans: '简体中文',
  },
  actions: {
    retry: '重试',
    reset: '恢复默认值',
    resetLocale: '将语言区域恢复为默认值',
    resetTimezone: '将时区恢复为默认值',
    save: '保存偏好',
  },
  messages: {
    saved: '偏好已保存',
    reset: '偏好已恢复默认值',
  },
  errors: {
    forbidden: '你无权修改这些偏好。',
    generic: '偏好更新失败。',
    invalidResponse: '设置服务返回了无效响应。',
    invalidValue: '请检查语言区域或 IANA 时区。',
    notFound: '此设置已不可用。',
    preconditionRequired: '请刷新页面后再修改此设置。',
    unavailable: '设置服务暂时不可用。',
    versionConflict: '此设置已在其他位置更新，现已载入最新值。',
  },
} as const satisfies LocaleMessageShape<SettingMessages>;

export default messages;
