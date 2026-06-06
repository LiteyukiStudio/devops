import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Box, KeyRound, LayoutDashboard, Link2, PanelLeftClose, PanelLeftOpen, Settings, ShieldCheck, Users } from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '../api/client'
import { usePublicConfig } from '../app/public-config-context'
import { useTheme } from '../app/theme-context'
import { AuthErrorPage } from '../components/common/auth-error-page'
import { PageMotion } from '../components/common/motion'
import { SidebarUserPanel } from '../components/common/sidebar-user-panel'
import { ThemeModeSegmented } from '../components/common/theme-mode-segmented'
import { Button } from '../components/ui/button'
import { cn } from '../lib/utils'

const navItems = [
  { to: '/projects', labelKey: 'projects', icon: LayoutDashboard },
  { to: '/access-tokens', labelKey: 'accessTokens', icon: KeyRound },
  { to: '/settings/security', labelKey: 'security', icon: Link2 },
  { to: '/settings/auth-providers', labelKey: 'authProviders', icon: ShieldCheck, permission: 'user.manage' },
  { to: '/settings/users', labelKey: 'users', icon: Users, permission: 'user.manage' },
  { to: '/settings/site', labelKey: 'siteSettings', icon: Settings },
]

export function AppLayout() {
  const { i18n, t } = useTranslation()
  const { mode, setMode } = useTheme()
  const configs = usePublicConfig()
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => window.localStorage.getItem('sidebar-collapsed') === 'true')
  const user = useQuery({ queryKey: ['current-user'], queryFn: api.getCurrentUser })
  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      queryClient.clear()
      navigate('/login')
    },
    onError: error => toast.error(error.message),
  })
  const updateLanguage = useMutation({
    mutationFn: api.updateCurrentUser,
    onSuccess: (result) => {
      i18n.changeLanguage(result.language)
      queryClient.setQueryData(['current-user'], result)
    },
    onError: error => toast.error(error.message),
  })

  const handleLogout = () => {
    logout.mutate()
  }

  const toggleSidebar = () => {
    setSidebarCollapsed((value) => {
      window.localStorage.setItem('sidebar-collapsed', String(!value))
      return !value
    })
  }

  useEffect(() => {
    if (user.data?.language && i18n.language !== user.data.language)
      i18n.changeLanguage(user.data.language)
  }, [i18n, user.data?.language])

  if (user.isError) {
    return (
      <AuthErrorPage
        title="需要登录"
        description="请先使用本地账号或已授权的 OIDC 账号登录平台。"
      />
    )
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <motion.aside
        animate={{ width: sidebarCollapsed ? 72 : 256 }}
        className="fixed inset-y-0 left-0 hidden grid-rows-[auto_1fr_auto] overflow-hidden border-r border-border bg-surface lg:grid"
        transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="flex h-16 items-center gap-3 border-b border-border px-3">
          <Link
            aria-label={configs['site.title'] || t('appName')}
            className={cn('flex min-w-0 flex-1 items-center gap-3', sidebarCollapsed && 'justify-center')}
            to="/projects"
          >
            <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
              {configs['site.logoUrl']
                ? <img alt="" className="size-6 rounded-sm object-contain" src={configs['site.logoUrl']} />
                : <Box size={18} />}
            </span>
            <AnimatePresence>
              {!sidebarCollapsed && (
                <motion.span
                  animate={{ opacity: 1, x: 0 }}
                  className="truncate font-semibold"
                  exit={{ opacity: 0, x: -4 }}
                  initial={{ opacity: 0, x: -4 }}
                  transition={{ duration: 0.16, ease: 'easeOut' }}
                >
                  {configs['site.title'] || t('appName')}
                </motion.span>
              )}
            </AnimatePresence>
          </Link>
          {!sidebarCollapsed && (
            <Button aria-label="折叠侧边栏" className="size-8 shrink-0 px-0" variant="ghost" onClick={toggleSidebar}>
              <PanelLeftClose size={16} />
            </Button>
          )}
        </div>
        <nav className="space-y-1 overflow-y-auto p-3">
          {navItems.filter(item => !item.permission || user.data?.permissions.includes(item.permission)).map(item => (
            <NavLink
              key={item.to}
              className={({ isActive }) => cn(
                'flex h-10 items-center gap-3 rounded-md px-3 text-sm text-muted-foreground transition-all duration-150 hover:bg-muted hover:text-foreground',
                sidebarCollapsed && 'justify-center px-0',
                isActive && 'bg-muted text-foreground',
              )}
              title={t(item.labelKey)}
              to={item.to}
            >
              <item.icon size={17} />
              <AnimatePresence>
                {!sidebarCollapsed && (
                  <motion.span
                    animate={{ opacity: 1, x: 0 }}
                    className="truncate"
                    exit={{ opacity: 0, x: -4 }}
                    initial={{ opacity: 0, x: -4 }}
                    transition={{ duration: 0.14, ease: 'easeOut' }}
                  >
                    {t(item.labelKey)}
                  </motion.span>
                )}
              </AnimatePresence>
            </NavLink>
          ))}
        </nav>
        <div className="grid gap-3 border-t border-border p-3">
          {sidebarCollapsed
            ? (
                <Button aria-label="展开侧边栏" className="size-9 px-0" variant="ghost" onClick={toggleSidebar}>
                  <PanelLeftOpen size={16} />
                </Button>
              )
            : (
                <>
                  <ThemeModeSegmented mode={mode} setMode={setMode} />
                  <div className="flex items-center gap-2">
                    <select
                      aria-label="语言"
                      className="h-9 flex-1 rounded-md border border-border bg-background px-2 text-sm transition duration-150 focus:border-primary"
                      value={user.data?.language || 'zh-CN'}
                      onChange={event => updateLanguage.mutate({ language: event.target.value as 'zh-CN' | 'en-US' })}
                    >
                      <option value="zh-CN">中文</option>
                      <option value="en-US">English</option>
                    </select>
                  </div>
                  <SidebarUserPanel
                    logoutLabel={t('logout')}
                    logoutPending={logout.isPending}
                    user={user.data}
                    onLogout={handleLogout}
                  />
                </>
              )}
        </div>
      </motion.aside>

      <div className={cn('transition-[padding] duration-200 ease-out lg:min-h-screen', sidebarCollapsed ? 'lg:pl-[72px]' : 'lg:pl-64')}>
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-border bg-background/90 px-4 backdrop-blur lg:px-6">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{configs['site.title'] || t('appName')}</p>
            <p className="text-xs text-muted-foreground lg:hidden">{user.data?.email ?? 'demo@liteyuki.dev'}</p>
          </div>
          <div className="flex items-center gap-2 lg:hidden">
            <select
              aria-label="语言"
              className="h-9 rounded-md border border-border bg-surface px-2 text-sm"
              value={user.data?.language || 'zh-CN'}
              onChange={event => updateLanguage.mutate({ language: event.target.value as 'zh-CN' | 'en-US' })}
            >
              <option value="zh-CN">中文</option>
              <option value="en-US">English</option>
            </select>
            <Button aria-label={t('logout')} disabled={logout.isPending} variant="ghost" onClick={handleLogout}>
              {t('logout')}
            </Button>
          </div>
        </header>
        <main className="mx-auto max-w-7xl px-4 py-6 lg:px-6">
          <AnimatePresence mode="wait">
            <PageMotion key={location.pathname}>
              <Outlet />
            </PageMotion>
          </AnimatePresence>
        </main>
      </div>
    </div>
  )
}
