// Errors translations - Simplified Chinese
const messages = {
  notFound: '页面未找到',
  serverError: '服务器错误',
  networkError: '网络错误，请检查网络连接',
  unauthorized: '请登录后继续',
  forbidden: '您没有权限访问此资源',
  authForbiddenDescription: '当前会话无法访问受保护应用。如非预期，请联系管理员。',
  authUnavailable: '暂时无法验证登录状态',
  authUnavailableDescription: '您的登录状态未被更改。请检查网络连接后重试。',
} as const;

export default messages;

export type ErrorsMessages = typeof messages;
