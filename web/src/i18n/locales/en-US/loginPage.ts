const loginPage = {
  subtitle: 'Sign in to the console with a local account',
  email: 'Email',
  emailHint: 'Local account email. Use an identity provider button below for OIDC login.',
  password: 'Password',
  passwordHint: 'Local account password. The default development account hint is only shown in development mode.',
  passwordRequired: 'Enter password',
  success: 'Signed in',
  devAccount: 'Development account: ',
  useProviderLogin: 'Sign in with {{provider}}',
  recentAccounts: 'Recent accounts',
  selectRecentAccount: 'Select {{name}} ({{email}})',
  oidcFallback: 'OIDC login failed. Try again later or contact the platform administrator.',
  oidcStateInvalid: 'Login state expired. Start OIDC login again.',
  oidcGroupDenied: 'This account is not in an allowed OIDC group. Contact the platform administrator.',
  oidcEmailRequired: 'OIDC account must provide a verified email.',
  oidcAdmissionDenied: 'This account is not invited and does not match allowed email domains or groups.',
  oidcProviderDisabled: 'This OIDC provider is disabled.',
  oidcBindFailed: 'Third-party login binding failed. Make sure this identity is not bound to another account.',
  authForbidden: 'This account does not have permission to perform this action.',
}

export default loginPage
