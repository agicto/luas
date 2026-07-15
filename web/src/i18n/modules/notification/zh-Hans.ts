const messages = {
  open: '打开通知中心',
  title: '通知',
  unreadCount: '{count} 条未读通知',
  markAllRead: '全部标为已读',
  empty: '暂无通知',
  loading: '正在加载通知',
  loadError: '无法加载通知，请重试。',
  retry: '重试',
  unread: '未读',
  preferences: '通知偏好',
  preferencesDescription: '这些设置只影响后续的非必达通知。',
  inApp: '站内通知',
  inAppDescription: '在 Luas 控制台通知中心显示新消息。',
  email: '邮件通知',
  emailDescription: '通过已配置的邮件供应商发送新消息。',
  cancel: '取消',
  save: '保存偏好',
  saved: '通知偏好已更新',
  saveError: '无法保存通知偏好。',
  readError: '无法更新通知已读状态。',
  markAllError: '无法将通知全部标为已读。',
} as const;

export default messages;
export type NotificationMessages = typeof messages;
