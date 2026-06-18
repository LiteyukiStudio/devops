const loginPage = {
  subtitle: '使用本地账号登录控制台',
  email: '邮箱',
  emailHint: '本地账号邮箱。OIDC 登录请使用下方身份源按钮。',
  password: '密码',
  passwordHint: '本地账号密码。开发模式默认账号提示只会在开发环境显示。',
  passwordRequired: '请输入密码',
  success: '登录成功',
  devAccount: '开发默认账号：',
  useProviderLogin: '使用 {{provider}} 登录',
  recentAccounts: '最近登录',
  selectRecentAccount: '选择 {{name}}（{{email}}）',
  oidcFallback: 'OIDC 登录失败，请稍后重试或联系平台管理员。',
  oidcStateInvalid: '登录状态已失效，请重新发起 OIDC 登录。',
  oidcGroupDenied: '当前账号未命中允许的 OIDC 组，请联系平台管理员。',
  oidcEmailRequired: 'OIDC 账号需要提供已验证邮箱。',
  oidcAdmissionDenied: '当前账号未被邀请，也不在允许的邮箱域或组中。',
  oidcProviderDisabled: '该 OIDC 身份源已被禁用。',
  oidcBindFailed: '第三方登录绑定失败，请确认该身份未绑定其他账号。',
  authForbidden: '当前账号没有执行此操作的权限。',
}

export default loginPage
