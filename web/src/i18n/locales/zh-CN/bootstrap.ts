const bootstrap = {
  title: '初始化',
  description: '创建第一个平台管理员账号。',
  email: '管理员邮箱',
  emailHint: '第一个平台管理员账号的登录邮箱，也是后续绑定 OIDC 时查找本地账号的依据。',
  name: '管理员名称',
  nameRequired: '请输入管理员名称',
  nameHint: '管理员在侧边栏和审计记录里显示的名称。',
  passwordHint: '本地管理员密码，至少 8 位。生产环境请使用强密码，并尽快配置 OIDC。',
  token: '初始化令牌',
  tokenHint: '由平台部署管理员通过服务端配置提供，只用于首次初始化。',
  tokenRequired: '请输入初始化令牌',
  create: '创建管理员',
  success: '平台管理员已初始化',
  note: '仅当平台没有任何 PlatformAdmin 时允许初始化。生产环境不会显示开发默认账号。',
}

export default bootstrap
