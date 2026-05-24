// Auth translations - Simplified Chinese
const messages = {
  // Page titles
  welcomeBack: '欢迎回来',
  signInToContinue: '登录以继续访问您的账户',
  createAccount: '创建账户',
  getStarted: '开始使用平台',
  registrationDisabled: '注册已关闭',
  registrationDisabledMessage: '该平台目前暂不开放注册，请联系管理员获取访问权限。',
  backToSignIn: '返回登录',
  
  // Form labels
  email: '邮箱',
  password: '密码',
  confirmPassword: '确认密码',
  fullName: '姓名',
  
  // Form placeholders
  enterEmail: '请输入邮箱地址',
  enterPassword: '请输入密码',
  createPassword: '创建一个强密码',
  confirmYourPassword: '请再次输入密码',
  enterFullName: '请输入您的姓名',
  
  // Actions
  login: '登录',
  logout: '退出登录',
  register: '注册',
  signIn: '登录',
  signUp: '注册',
  forgotPassword: '忘记密码？',
  demoCredentials: '演示账号',
  demoCredentialsHint: '使用默认账号',
  demoCredentialsValue: 'admin@example.com / admin123',
  
  // Links
  noAccount: '没有账号？',
  hasAccount: '已有账号？',
  rememberMe: '记住我',
  
  // Terms
  agreeToTerms: '我同意',
  termsOfService: '服务条款',
  and: '和',
  privacyPolicy: '隐私政策',
  orContinueWith: '或通过以下方式继续',
  
  // Password requirements
  passwordRequirements: '密码要求',
  passwordReqLength: '至少8个字符',
  passwordReqCase: '包含大小写字母',
  passwordReqNumber: '包含至少一个数字',
  passwordReqSpecial: '包含至少一个特殊字符',
  
  // Decorative panel
  heroTitle: '构建卓越产品',
  heroSubtitle: '现代化Web脚手架，助力快速可靠的产品交付',
  feature1: '极速开发体验',
  feature2: '企业级安全保障',
  feature3: '可扩展架构',
  feature4: '全天候技术支持',
  
  // Validation messages
  nameRequired: '请输入姓名',
  nameMinLength: '姓名至少需要2个字符',
  nameTooLong: '姓名过长',
  emailRequired: '请输入邮箱',
  emailInvalid: '请输入有效的邮箱地址',
  passwordRequired: '请输入密码',
  passwordMinLength: '密码至少需要8个字符',
  passwordTooLong: '密码过长',
  passwordTooShort: '密码过短',
  passwordInvalid: '密码必须包含大小写字母、数字和特殊字符',
  confirmPasswordRequired: '请确认密码',
  passwordsDoNotMatch: '两次输入的密码不一致',
  termsRequired: '请同意服务条款和隐私政策',
  
  // Toast messages
  loginSuccess: '登录成功',
  loginFailed: '登录失败',
  welcomeBackUser: '欢迎回来，{name}',
  logoutSuccess: '退出成功',
  logoutFailed: '退出失败',
  registerSuccess: '注册成功',
  registerFailed: '注册失败',
  accountCreated: '账号创建成功，正在登录...',
  invalidCredentials: '邮箱或密码错误',
  
  // Social login
  signInWithGoogle: '使用 Google 登录',
  signInWithApple: '使用 Apple 登录',
  signInWithGithub: '使用 GitHub 登录',
  signUpWithGoogle: '使用 Google 注册',
  signUpWithApple: '使用 Apple 注册',
  signUpWithGithub: '使用 GitHub 注册',
  
  // Footer
  allRightsReserved: '版权所有',
};

export default messages;

export type AuthMessages = typeof messages;
