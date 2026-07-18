import type { ReactNode } from 'react'
import type { DashboardActivity, DashboardAttentionItem, DashboardProjectShortcut, DashboardReadinessItem } from '@/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Activity, AppWindow, Boxes, CheckCircle2, Container, FolderKanban, Globe2, Hammer, Pin, Rocket, Server, ShieldAlert, ShieldCheck, Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { api } from '@/api'
import { ErrorState } from '@/components/common/error-state'
import { StatusBadge, StatusValueBadge } from '@/components/common/status-badge'
import { formatCompactDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { WORKFLOW_STATUS_REFETCH_INTERVAL_MS } from '@/lib/polling'

export function DashboardPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const dashboard = useQuery({
    queryKey: ['dashboard'],
    queryFn: api.getDashboard,
    refetchInterval: WORKFLOW_STATUS_REFETCH_INTERVAL_MS,
  })
  const toggleProjectPin = useMutation<void, Error, { pinned: boolean, projectId: string }>({
    mutationFn: async ({ pinned, projectId }) => {
      if (pinned)
        await api.unpinProject(projectId)
      else
        await api.pinProject(projectId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['project-pins'] })
    },
  })

  if (dashboard.isError) {
    return (
      <ErrorState
        description={t('dashboardPage.loadFailedDescription')}
        title={t('dashboardPage.loadFailedTitle')}
      />
    )
  }

  if (!dashboard.data) {
    return <Card className="p-4 text-sm text-muted-foreground">{t('common.loading')}</Card>
  }

  const overview = dashboard.data
  const activeTasks = overview.summary.activeBuilds + overview.summary.activeReleases
  const hasMoreProjects = overview.summary.projects > overview.projects.length

  return (
    <div className="grid min-w-0 gap-4">
      <section className="min-w-0 max-w-full">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <SectionTitle icon={<FolderKanban size={18} />} title={t('dashboardPage.projectShortcuts')} />
          {hasMoreProjects && (
            <Link className="text-sm font-medium text-muted-foreground transition hover:text-primary" to="/projects">
              {t('dashboardPage.viewAllProjects')}
            </Link>
          )}
        </div>
        {overview.projects.length
          ? (
              <div className="mt-3 min-w-0 max-w-full overflow-x-auto overflow-y-hidden pb-2">
                <div className="inline-flex min-w-max gap-3">
                  {overview.projects.map(project => (
                    <ProjectShortcutCard
                      key={project.id}
                      isPinPending={toggleProjectPin.isPending}
                      project={project}
                      onTogglePin={(projectId, pinned) => toggleProjectPin.mutate({ pinned, projectId })}
                    />
                  ))}
                </div>
              </div>
            )
          : <p className="py-4 text-sm text-muted-foreground">{t('projectSpaces.emptyTitle')}</p>}
      </section>

      <section className="grid gap-3">
        <h2 className="sr-only">{t('dashboardPage.workOverview')}</h2>
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge tone={overview.summary.attentionItems ? 'warning' : 'success'}>
            {overview.summary.attentionItems ? t('dashboardPage.needsAttention') : t('dashboardPage.healthy')}
          </StatusBadge>
          <StatusBadge>{t('dashboardPage.resourceTotals', { applications: overview.summary.applications, projects: overview.summary.projects })}</StatusBadge>
          {activeTasks > 0 && (
            <span className="text-xs text-muted-foreground">{t('dashboardPage.activeTasksTotal', { count: activeTasks })}</span>
          )}
        </div>
        <div className="grid gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-2 xl:grid-cols-4">
          <DashboardMetric icon={<Hammer size={18} />} label={t('dashboardPage.activeBuilds')} to="/events?categories=build&statuses=in_progress" value={overview.summary.activeBuilds} />
          <DashboardMetric icon={<Rocket size={18} />} label={t('dashboardPage.activeReleases')} to="/events?categories=release&statuses=in_progress" value={overview.summary.activeReleases} />
          <DashboardMetric icon={<ShieldAlert size={18} />} label={t('dashboardPage.attentionItems')} tone={overview.summary.attentionItems ? 'danger' : 'neutral'} to="/events?severities=error&severities=warning" value={overview.summary.attentionItems} />
          <DashboardMetric icon={<Server size={18} />} label={t('dashboardPage.healthyClusters')} tone={overview.summary.healthyClusters < overview.summary.totalClusters ? 'warning' : 'neutral'} to="/clusters" value={`${overview.summary.healthyClusters}/${overview.summary.totalClusters}`} />
        </div>
        <AttentionPanel items={overview.attention} />
      </section>

      <Card className="grid min-w-0 overflow-hidden p-0 xl:grid-cols-3">
        <section className="min-w-0 p-4 xl:col-span-2">
          <div className="flex items-center justify-between gap-3">
            <SectionTitle icon={<Activity size={18} />} title={t('dashboardPage.recentActivity')} />
            <Link className="text-sm font-medium text-muted-foreground transition hover:text-primary" to="/events">
              {t('dashboardPage.viewAllEvents')}
            </Link>
          </div>
          <div className="mt-3 h-72 overflow-y-auto pr-1">
            {overview.activities.length
              ? (
                  <div className="divide-y divide-border">
                    {overview.activities.map(activity => <ActivityRow key={activity.id} activity={activity} />)}
                  </div>
                )
              : <p className="py-4 text-sm text-muted-foreground">{t('dashboardPage.noActivity')}</p>}
          </div>
        </section>

        <section className="border-t border-border p-4 xl:border-l xl:border-t-0">
          <SectionTitle icon={<Boxes size={18} />} title={t('dashboardPage.platformReadiness')} />
          <div className="mt-3 grid gap-2">
            <ReadinessRow icon={<Container size={16} />} item={overview.readiness.registries} kind="registries" label={t('registries')} to="/registries" />
            <ReadinessRow icon={<Server size={16} />} item={overview.readiness.clusters} kind="clusters" label={t('clusters')} to="/clusters" />
          </div>
        </section>
      </Card>
    </div>
  )
}

function ProjectShortcutCard({ isPinPending, onTogglePin, project }: { isPinPending: boolean, onTogglePin: (projectId: string, pinned: boolean) => void, project: DashboardProjectShortcut }) {
  const { t } = useTranslation()
  return (
    <Link
      className="group relative grid min-h-28 w-64 flex-none gap-3 rounded-md border border-border bg-background p-3 transition-all duration-150 hover:border-primary/50 hover:bg-muted/35"
      to={`/projects/${project.id}`}
    >
      <div className="min-w-0">
        <span className="block truncate pr-9 font-medium">{project.name}</span>
        <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{project.description || t('common.noDescription')}</p>
      </div>
      <Button
        aria-label={project.pinned ? t('common.unpinProject') : t('common.pinProject')}
        className={`absolute right-2 top-2 size-8 transition-opacity ${project.pinned ? 'text-primary opacity-100 hover:text-primary' : 'opacity-0 group-hover:opacity-100 focus-visible:opacity-100'}`}
        disabled={isPinPending}
        size="icon"
        type="button"
        variant="ghost"
        onClick={(event) => {
          event.preventDefault()
          event.stopPropagation()
          onTogglePin(project.id, project.pinned)
        }}
      >
        <Pin className={`size-4 ${project.pinned ? 'fill-current' : ''}`} />
      </Button>
      <div className="flex min-w-0 flex-wrap items-center gap-2 self-end">
        <StatusBadge>{t('dashboardPage.appsCount', { count: project.applicationCount })}</StatusBadge>
        {project.latestActivity
          ? <StatusValueBadge labelKeyPrefix="eventsPage.statuses" value={project.latestActivity.status} />
          : <StatusBadge tone="neutral">{t('dashboardPage.noActivityShort')}</StatusBadge>}
        {project.latestActivity && <span className="text-xs text-muted-foreground">{formatCompactDateTime(project.latestActivity.occurredAt)}</span>}
      </div>
    </Link>
  )
}

function DashboardMetric({ icon, label, to, tone = 'neutral', value }: { icon: ReactNode, label: string, to: string, tone?: 'danger' | 'neutral' | 'warning', value: number | string }) {
  const toneClass = tone === 'danger' ? 'text-red-600 dark:text-red-400' : tone === 'warning' ? 'text-amber-700 dark:text-amber-400' : ''
  return (
    <Link className={`group bg-surface p-3 transition-colors hover:bg-muted/40 ${toneClass}`} to={to}>
      <div className="flex items-center gap-2 text-sm text-muted-foreground transition-colors group-hover:text-primary">
        {icon}
        <span>{label}</span>
      </div>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </Link>
  )
}

function AttentionPanel({ items }: { items: DashboardAttentionItem[] }) {
  const { t } = useTranslation()
  return (
    <div className="grid gap-2 border-t border-border pt-3 lg:grid-cols-4 lg:items-start lg:gap-3">
      <div className="flex min-h-8 items-center gap-2 text-sm font-medium">
        <ShieldAlert size={16} />
        {t('dashboardPage.attention')}
      </div>
      {items.length
        ? (
            <div className="flex min-w-0 flex-wrap gap-2 lg:col-span-3">
              {items.slice(0, 4).map(item => (
                <Link key={item.key} className="group flex min-h-8 min-w-0 max-w-full items-center gap-2 rounded-md bg-muted/40 px-2 py-1.5 transition-colors hover:bg-muted" to={activityTarget(item.latest)}>
                  <span className="shrink-0 text-muted-foreground transition-colors group-hover:text-primary">{categoryIcon(item.category)}</span>
                  <span className="truncate text-sm">{eventTypeLabel(t, item.latest.type)}</span>
                  {item.occurrences > 1 && <StatusBadge tone={item.severity === 'error' ? 'danger' : 'warning'}>{t('dashboardPage.occurrences', { count: item.occurrences })}</StatusBadge>}
                </Link>
              ))}
              {items.length > 4 && <Link className="flex min-h-8 items-center px-2 text-sm font-medium text-primary" to="/events?severities=error&severities=warning">{t('dashboardPage.moreAttention', { count: items.length - 4 })}</Link>}
            </div>
          )
        : (
            <div className="flex min-h-8 items-center gap-2 text-sm text-muted-foreground lg:col-span-3">
              <CheckCircle2 className="text-emerald-600" size={16} />
              {t('dashboardPage.noIssues')}
            </div>
          )}
    </div>
  )
}

function ActivityRow({ activity }: { activity: DashboardActivity }) {
  const { t } = useTranslation()
  return (
    <Link className="group grid gap-2 py-3 transition-colors first:pt-0 hover:text-primary sm:flex sm:items-center sm:justify-between" to={activityTarget(activity)}>
      <div className="flex min-w-0 flex-1 items-start gap-3">
        <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
          {categoryIcon(activity.category)}
        </div>
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="truncate font-medium">{eventTypeLabel(t, activity.type)}</span>
            <StatusValueBadge labelKeyPrefix="eventsPage.statuses" value={activity.status} />
          </div>
          <p className="mt-1 truncate text-sm text-muted-foreground">
            {activityContext(activity) || activity.message || t('eventsPage.noMessage')}
          </p>
        </div>
      </div>
      <span className="pl-11 text-xs text-muted-foreground sm:pl-0">{formatCompactDateTime(activity.occurredAt)}</span>
    </Link>
  )
}

function ReadinessRow({ icon, item, kind, label, to }: { icon: ReactNode, item: DashboardReadinessItem, kind: 'clusters' | 'registries', label: string, to: string }) {
  const { t } = useTranslation()
  const value = kind === 'clusters' ? `${item.available}/${item.total}` : item.total
  return (
    <Link className="group flex items-center justify-between gap-3 rounded-md bg-muted/30 px-3 py-3 transition-colors hover:bg-muted" to={to}>
      <div className="flex min-w-0 items-center gap-2 text-sm font-medium">
        <span className="text-muted-foreground">{icon}</span>
        <span className="truncate">{label}</span>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <StatusValueBadge labelKeyPrefix="dashboardPage.readinessStatuses" value={item.status} />
        <span className="text-sm tabular-nums text-muted-foreground" title={t('dashboardPage.availableCount')}>{value}</span>
      </div>
    </Link>
  )
}

function SectionTitle({ icon, title }: { icon: ReactNode, title: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground">{icon}</span>
      <h3 className="text-base font-semibold">{title}</h3>
    </div>
  )
}

function eventTypeLabel(t: ReturnType<typeof useTranslation>['t'], type: string) {
  return t(`eventsPage.types.${type.replaceAll('.', '_')}`, { defaultValue: type })
}

function activityContext(activity: DashboardActivity) {
  return [activity.project?.name, activity.application?.name, activity.deploymentTarget?.name].filter(Boolean).join(' · ')
}

function activityTarget(activity: DashboardActivity) {
  const primary = activity.links.primary
  if (primary?.startsWith('/'))
    return primary
  if (activity.project && activity.application) {
    const tab = activity.category === 'build' ? 'builds' : activity.category === 'gateway' || activity.category === 'certificate' ? 'gateway' : 'deployments'
    return `/projects/${activity.project.id}/apps/${activity.application.id}#tab=${tab}`
  }
  if (activity.project)
    return `/projects/${activity.project.id}`
  return '/events'
}

function categoryIcon(category: string) {
  const className = 'size-4'
  if (category === 'build')
    return <Hammer className={className} />
  if (category === 'release')
    return <Rocket className={className} />
  if (category === 'hook')
    return <Workflow className={className} />
  if (category === 'gateway')
    return <Globe2 className={className} />
  if (category === 'certificate')
    return <ShieldCheck className={className} />
  if (category === 'application')
    return <AppWindow className={className} />
  return <Activity className={className} />
}
