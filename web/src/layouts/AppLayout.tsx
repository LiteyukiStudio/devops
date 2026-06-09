import { useQuery } from '@tanstack/react-query'
import { Container, FolderKanban, GitBranch, LayoutDashboard, Link2, Server, Settings, ShieldCheck, Users } from 'lucide-react'
import { AnimatePresence } from 'motion/react'
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, NavLink, Outlet, useLocation } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { useDocumentTitle } from '@/app/document-title'
import { usePublicConfig } from '@/app/public-config-context'
import { useSession } from '@/app/session-context'
import { useTheme } from '@/app/theme-context'
import { DebugFloatingPanel } from '@/components/common/debug-floating-panel'
import { PageMotion } from '@/components/common/motion'
import { SidebarUserPanel } from '@/components/common/sidebar-user-panel'
import { ThemeModeSegmented } from '@/components/common/theme-mode-segmented'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { cn } from '@/lib/utils'

const navSections = [
  {
    titleKey: 'DevOps',
    items: [
      { to: '/code-repositories', labelKey: 'codeRepositories', icon: GitBranch },
      { to: '/registries', labelKey: 'registries', icon: Container },
      { to: '/clusters', labelKey: 'clusters', icon: Server },
    ],
  },
  {
    titleKey: 'nav.personalWorkspace',
    items: [
      { to: '/settings/account', labelKey: 'account', icon: Link2 },
    ],
  },
  {
    titleKey: 'nav.systemManagement',
    items: [
      { to: '/settings/auth-providers', labelKey: 'authProviders', icon: ShieldCheck, permission: 'user.manage' },
      { to: '/settings/users', labelKey: 'users', icon: Users, permission: 'user.manage' },
      { to: '/settings/site', labelKey: 'siteSettings', icon: Settings },
    ],
  },
]

const pageMetaRules = [
  { match: (pathname: string) => pathname === '/dashboard', titleKey: 'dashboard', descriptionKey: 'dashboardPage.description' },
  { match: (pathname: string) => /^\/projects\/[^/]+\/apps\/[^/]+$/.test(pathname), titleKey: 'apps.detailTitle', descriptionKey: 'apps.detailDescription' },
  { match: (pathname: string) => /^\/projects\/[^/]+\/repositories$/.test(pathname), titleKey: 'repositories.title', descriptionKey: 'repositories.description' },
  { match: (pathname: string) => /^\/projects\/[^/]+\/members$/.test(pathname), titleKey: 'projectMembers.title', descriptionKey: 'projectMembers.description' },
  { match: (pathname: string) => /^\/projects\/[^/]+\/apps$/.test(pathname), titleKey: 'apps.title', descriptionKey: 'apps.description' },
  { match: (pathname: string) => /^\/projects\/[^/]+$/.test(pathname), titleKey: 'projectSpaces.workspaceTitle', descriptionKey: 'projectSpaces.workspaceDescription' },
  { match: (pathname: string) => pathname === '/projects', titleKey: 'projectSpaces.title', descriptionKey: 'projectSpaces.description' },
  { match: (pathname: string) => pathname === '/code-repositories', titleKey: 'codeRepositories', descriptionKey: 'codeRepositoriesPage.description' },
  { match: (pathname: string) => pathname === '/registries', titleKey: 'registries', descriptionKey: 'registriesPage.description' },
  { match: (pathname: string) => pathname === '/clusters', titleKey: 'clusters', descriptionKey: 'clustersPage.description' },
  { match: (pathname: string) => pathname === '/settings/account' || pathname === '/settings/security', titleKey: 'account', descriptionKey: 'accountPage.description' },
  { match: (pathname: string) => pathname === '/settings/auth-providers', titleKey: 'authProvidersPage.title', descriptionKey: 'authProvidersPage.description' },
  { match: (pathname: string) => pathname === '/settings/users', titleKey: 'usersPage.title', descriptionKey: 'usersPage.description' },
  { match: (pathname: string) => pathname === '/settings/site', titleKey: 'siteSettings', descriptionKey: 'settings.siteDescription' },
]

function sidebarMenuButtonClassName(active?: boolean) {
  return cn(
    'flex h-10 w-full min-w-0 max-w-full items-center gap-3 overflow-hidden rounded-full px-3 text-sm font-normal leading-none text-muted-foreground transition-all duration-150 hover:bg-muted hover:text-foreground',
    active && 'bg-muted text-foreground',
  )
}

export function AppLayout() {
  const { i18n, t } = useTranslation()
  const { mode, setMode } = useTheme()
  const { isLoading: sessionLoading, isLoggingOut, logout, user } = useSession()
  const configs = usePublicConfig()
  const location = useLocation()
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects, enabled: Boolean(user) })
  const projectRouteMatch = location.pathname.match(/^\/projects\/([^/]+)/)
  const appRouteMatch = location.pathname.match(/^\/projects\/([^/]+)\/apps\/([^/]+)$/)
  const currentProject = useQuery({
    queryKey: ['project', projectRouteMatch?.[1]],
    queryFn: () => api.getProject(projectRouteMatch?.[1] ?? ''),
    enabled: Boolean(user && projectRouteMatch),
  })
  const currentApplication = useQuery({
    queryKey: ['application', appRouteMatch?.[1], appRouteMatch?.[2]],
    queryFn: () => api.getApplication(appRouteMatch?.[1] ?? '', appRouteMatch?.[2] ?? ''),
    enabled: Boolean(user && appRouteMatch),
  })
  const pageMeta = useMemo(() => {
    const rule = pageMetaRules.find(item => item.match(location.pathname))
    const projectWorkspaceMatch = location.pathname.match(/^\/projects\/([^/]+)$/)
    const project = currentProject.data ?? (projectRouteMatch ? projects.data?.find(project => project.id === projectRouteMatch[1]) : undefined)
    const application = appRouteMatch ? currentApplication.data : undefined
    return {
      description: projectWorkspaceMatch && project ? project.description || t('common.noDescription') : rule?.descriptionKey ? t(rule.descriptionKey) : t('common.noDescription'),
      title: application
        ? t('apps.detailTopbarTitle', { name: application.name, projectName: project?.name ?? t('projectSpaces.title') })
        : projectWorkspaceMatch && project
          ? t('projectSpaces.detailTopbarTitle', { name: project.name })
          : rule?.titleKey ? t(rule.titleKey) : configs['site.title'] || t('appName'),
    }
  }, [appRouteMatch, configs, currentApplication.data, currentProject.data, location.pathname, projectRouteMatch, projects.data, t])
  useDocumentTitle(pageMeta.title)
  const pageMotionKey = /^\/projects\/[^/]+$/.test(location.pathname) ? '/projects/:projectId' : location.pathname
  const handleLogout = () => {
    logout().catch(error => toast.error(error.message))
  }

  useEffect(() => {
    if (user?.language && i18n.language !== user.language)
      i18n.changeLanguage(user.language)
  }, [i18n, user?.language])

  if (sessionLoading) {
    return (
      <div className="grid min-h-screen place-items-center bg-background text-sm text-muted-foreground">
        {t('common.loading')}
      </div>
    )
  }

  if (!user) {
    const redirect = `${location.pathname}${location.search}`
    return <Navigate to={`/login?redirect=${encodeURIComponent(redirect)}`} replace />
  }

  return (
    <div className="h-dvh overflow-hidden bg-background text-foreground">
      <div className="flex h-full w-full min-w-0 overflow-hidden">
        <Sidebar>
          <SidebarHeader>
            <Link
              aria-label={configs['site.title'] || t('appName')}
              className="flex h-16 w-full min-w-0 max-w-full items-center gap-3 overflow-hidden px-5"
              to="/projects"
            >
              <img
                alt=""
                className="size-10 shrink-0 rounded-xl object-contain"
                src={configs['site.logoUrl'] || '/liteyuki-logo.svg'}
              />
              <span className="min-w-0 flex-1 truncate font-semibold">{configs['site.title'] || t('appName')}</span>
            </Link>
          </SidebarHeader>
          <SidebarContent>
            {navSections.map((section, index) => {
              const items = section.items.filter(item => !item.permission || user.permissions.includes(item.permission))
              if (items.length === 0)
                return null

              return (
                <SidebarGroup key={section.titleKey} className={index > 0 ? 'mt-4' : undefined}>
                  {index > 0 && <Separator className="mb-4" />}
                  <SidebarGroupLabel>{section.titleKey === 'DevOps' ? section.titleKey : t(section.titleKey)}</SidebarGroupLabel>
                  <SidebarMenu>
                    {section.titleKey === 'DevOps' && (
                      <SidebarMenuItem>
                        <NavLink
                          className={({ isActive }) => sidebarMenuButtonClassName(isActive)}
                          title={t('dashboard')}
                          to="/dashboard"
                        >
                          <LayoutDashboard className="size-[17px] shrink-0" />
                          <span className="min-w-0 flex-1 truncate text-sm font-normal leading-none">{t('dashboard')}</span>
                        </NavLink>
                      </SidebarMenuItem>
                    )}
                    {section.titleKey === 'DevOps' && (
                      <SidebarMenuItem>
                        <NavLink
                          className={({ isActive }) => sidebarMenuButtonClassName(isActive || location.pathname.startsWith('/projects/'))}
                          title={t('projects')}
                          to="/projects"
                        >
                          <FolderKanban className="size-[17px] shrink-0" />
                          <span className="min-w-0 flex-1 truncate text-sm font-normal leading-none">{t('projects')}</span>
                        </NavLink>
                      </SidebarMenuItem>
                    )}
                    {items.map(item => (
                      <SidebarMenuItem key={item.to}>
                        <NavLink
                          className={({ isActive }) => sidebarMenuButtonClassName(isActive)}
                          title={t(item.labelKey)}
                          to={item.to}
                        >
                          <item.icon className="size-[17px] shrink-0" />
                          <span className="min-w-0 flex-1 truncate text-sm font-normal leading-none">{t(item.labelKey)}</span>
                        </NavLink>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroup>
              )
            })}
          </SidebarContent>
          <SidebarFooter>
            <ThemeModeSegmented mode={mode} setMode={setMode} />
            <SidebarUserPanel
              logoutLabel={t('logout')}
              logoutPending={isLoggingOut}
              user={user}
              onLogout={handleLogout}
            />
          </SidebarFooter>
        </Sidebar>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          <header className="z-10 flex min-h-20 shrink-0 items-center justify-between border-b border-border bg-background/90 px-4 py-3 backdrop-blur lg:px-5 xl:px-6">
            <div className="min-w-0">
              <h1 className="truncate text-xl font-semibold tracking-normal">{pageMeta.title}</h1>
              <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">{pageMeta.description}</p>
            </div>
            <div className="flex items-center gap-2 lg:hidden">
              <Button aria-label={t('logout')} disabled={isLoggingOut} variant="ghost" onClick={handleLogout}>
                {t('logout')}
              </Button>
            </div>
          </header>
          <main className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-4 py-5 lg:px-5 xl:px-6">
            <AnimatePresence mode="wait">
              <PageMotion key={pageMotionKey}>
                <Outlet />
              </PageMotion>
            </AnimatePresence>
          </main>
        </div>
      </div>
      <DebugFloatingPanel />
    </div>
  )
}
