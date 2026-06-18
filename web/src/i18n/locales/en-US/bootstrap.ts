const bootstrap = {
  title: 'Bootstrap',
  description: 'Create the first platform administrator account.',
  email: 'Administrator email',
  emailHint: 'Login email for the first platform administrator. Also used to match local accounts during later OIDC binding.',
  name: 'Administrator name',
  nameRequired: 'Enter administrator name',
  nameHint: 'Displayed in the sidebar and audit logs.',
  passwordHint: 'Local administrator password, at least 8 characters. Use a strong password in production and configure OIDC soon.',
  create: 'Create administrator',
  success: 'Platform administrator initialized',
  note: 'Initialization is allowed only when there is no PlatformAdmin. Development default account is not shown in production.',
}

export default bootstrap
