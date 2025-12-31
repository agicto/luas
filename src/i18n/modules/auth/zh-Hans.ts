// Auth translations - Simplified Chinese
const messages = {
  login: '登录',
  logout: '退出登录',
  register: '注册',
  email: '邮箱',
  password: '密码',
  confirmPassword: '确认密码',
  forgotPassword: '忘记密码？',
  rememberMe: '记住我',
  noAccount: '没有账号？',
  hasAccount: '已有账号？',
  
  // Toast messages
  loginSuccess: '登录成功',
  loginFailed: '登录失败',
  welcomeBack: '欢迎回来，{name}',
  logoutSuccess: '退出成功',
  logoutFailed: '退出失败',
  registerSuccess: '注册成功',
  registerFailed: '注册失败',
  accountCreated: '账号创建成功，正在登录...',
  invalidCredentials: '邮箱或密码错误',
};

export default messages;

export type AuthMessages = typeof messages;
