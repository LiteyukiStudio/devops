import type { ReactNode } from 'react'
import type { PlatformEvent, PlatformEventSnapshot } from '@/api'
import type { DataListColumn } from '@/components/common/data-list'
import { useQuery } from '@tanstack/react-query'
import { Activity, ExternalLink, Eye, Globe2, Hammer, RefreshCw, Rocket, ShieldCheck, Workflow } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { api } from '@/api'
import { useSession } from '@/app/session-context'
import { DataList } from '@/components/common/data-list'
import { ErrorState } from '@/components/common/error-state'
import { StatusValueBadge } from '@/components/common/status-badge'
import { formatAbsoluteDateTime, formatSmartDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'

const allValue = '__all__'

export function EventsPage() {
  const { t } = useTranslation()
  const { user } = useSession()
  const [searchParams] = useSearchParams()
  const isPlatformAdmin = user?.role === 'platform_admin'
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [search, setSearch] = useState('')
  const [scope, setScope] = useState<'mine' | 'all'>('mine')
  const [projectId, setProjectId] = useState(() => searchParams.get('projectId') ?? '')
  const [applicationId, setApplicationId] = useState(() => searchParams.get('applicationId') ?? '')
  const [deploymentTargetId, setDeploymentTargetId] = useState(() => searchParams.get('deploymentTargetId') ?? '')
  const [category, setCategory] = useState('')
  const [eventType, setEventType] = useState('')
  const [severity, setSeverity] = useState('')
  const [status, setStatus] = useState('')
  const [dateFrom, setDateFrom] = useState(() => dateDaysAgo(7))
  const [dateTo, setDateTo] = useState(() => dateDaysAgo(0))
  const [selectedEventId, setSelectedEventId] = useState('')

  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
  const applications = useQuery({
    queryKey: ['applications', projectId],
    queryFn: () => api.listApplications(projectId),
    enabled: Boolean(projectId),
  })
  const deploymentTargets = useQuery({
    queryKey: ['deployment-targets', projectId, applicationId],
    queryFn: () => api.listDeploymentTargets(projectId, applicationId),
    enabled: Boolean(projectId && applicationId),
  })
  const catalog = useQuery({ queryKey: ['platform-event-catalog'], queryFn: api.listPlatformEventCatalog })
  const events = useQuery({
    queryKey: ['platform-events', page, pageSize, search, scope, projectId, applicationId, deploymentTargetId, category, eventType, severity, status, dateFrom, dateTo],
    queryFn: () => api.listPlatformEvents({
      page,
      pageSize,
      search: search || undefined,
      sortBy: 'occurredAt',
      sortOrder: 'desc',
      scope: isPlatformAdmin ? scope : 'mine',
      projectId: projectId || undefined,
      applicationId: applicationId || undefined,
      deploymentTargetId: deploymentTargetId || undefined,
      category: category || undefined,
      type: eventType || undefined,
      severity: severity || undefined,
      status: status || undefined,
      dateFrom: dateFrom || undefined,
      dateTo: dateTo || undefined,
    }),
  })
  const selectedEvent = useQuery({
    queryKey: ['platform-event', selectedEventId],
    queryFn: () => api.getPlatformEvent(selectedEventId),
    enabled: Boolean(selectedEventId),
  })

  const categories = useMemo(() => [...new Set((catalog.data ?? []).map(item => item.category))], [catalog.data])
  const resetPage = () => setPage(1)
  const columns = useMemo<DataListColumn<PlatformEvent>[]>(() => [
    {
      key: 'event',
      header: t('eventsPage.columns.event'),
      width: 'primary',
      render: event => (
        <div className="flex min-w-0 items-start gap-3">
          <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            {categoryIcon(event.category)}
          </div>
          <div className="min-w-0">
            <p className="truncate font-medium">{eventTypeLabel(t, event.type)}</p>
            <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{event.message || t('eventsPage.noMessage')}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'resource',
      header: t('eventsPage.columns.resource'),
      width: 'normal',
      render: event => <EventResource event={event} />,
    },
    {
      key: 'severity',
      header: t('eventsPage.columns.severity'),
      width: 'status',
      render: event => <StatusValueBadge labelKeyPrefix="eventsPage.severities" value={event.severity} />,
    },
    {
      key: 'status',
      header: t('eventsPage.columns.status'),
      width: 'status',
      render: event => <StatusValueBadge labelKeyPrefix="eventsPage.statuses" value={event.status} />,
    },
    {
      key: 'occurredAt',
      header: t('eventsPage.columns.time'),
      width: 'compact',
      render: event => <span className="whitespace-nowrap text-sm text-muted-foreground">{formatSmartDateTime(event.occurredAt, t)}</span>,
    },
    {
      key: 'actions',
      header: t('common.actions'),
      sticky: 'right',
      width: 'actions',
      render: event => (
        <Button aria-label={t('eventsPage.viewDetails')} size="icon" variant="ghost" onClick={() => setSelectedEventId(event.id)}>
          <Eye className="size-4" />
        </Button>
      ),
    },
  ], [t])

  if (events.isError) {
    return (
      <ErrorState
        description={t('eventsPage.loadFailedDescription')}
        title={t('eventsPage.loadFailedTitle')}
      />
    )
  }

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          {isPlatformAdmin && (
            <EventFilterSelect
              label={t('eventsPage.filters.scope')}
              value={scope}
              onChange={(value) => {
                setScope(value as 'mine' | 'all')
                resetPage()
              }}
            >
              <SelectItem value="mine">{t('eventsPage.scopes.mine')}</SelectItem>
              <SelectItem value="all">{t('eventsPage.scopes.all')}</SelectItem>
            </EventFilterSelect>
          )}
          <EventFilterSelect
            label={t('eventsPage.filters.project')}
            value={projectId || allValue}
            onChange={(value) => {
              setProjectId(value === allValue ? '' : value)
              setApplicationId('')
              setDeploymentTargetId('')
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allProjects')}</SelectItem>
            {(projects.data ?? []).map(project => <SelectItem key={project.id} value={project.id}>{project.name}</SelectItem>)}
          </EventFilterSelect>
          <EventFilterSelect
            disabled={!projectId}
            label={t('eventsPage.filters.application')}
            value={applicationId || allValue}
            onChange={(value) => {
              setApplicationId(value === allValue ? '' : value)
              setDeploymentTargetId('')
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allApplications')}</SelectItem>
            {(applications.data ?? []).map(application => <SelectItem key={application.id} value={application.id}>{application.name}</SelectItem>)}
          </EventFilterSelect>
          <EventFilterSelect
            disabled={!applicationId}
            label={t('eventsPage.filters.deploymentTarget')}
            value={deploymentTargetId || allValue}
            onChange={(value) => {
              setDeploymentTargetId(value === allValue ? '' : value)
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allDeploymentTargets')}</SelectItem>
            {(deploymentTargets.data ?? []).map(target => <SelectItem key={target.id} value={target.id}>{target.name}</SelectItem>)}
          </EventFilterSelect>
          <EventFilterSelect
            label={t('eventsPage.filters.category')}
            value={category || allValue}
            onChange={(value) => {
              setCategory(value === allValue ? '' : value)
              setEventType('')
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allCategories')}</SelectItem>
            {categories.map(value => <SelectItem key={value} value={value}>{t(`eventsPage.categories.${value}`, { defaultValue: value })}</SelectItem>)}
          </EventFilterSelect>
          <EventFilterSelect
            label={t('eventsPage.filters.type')}
            value={eventType || allValue}
            onChange={(value) => {
              setEventType(value === allValue ? '' : value)
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allTypes')}</SelectItem>
            {(catalog.data ?? []).filter(item => !category || item.category === category).map(item => (
              <SelectItem key={item.type} value={item.type}>{eventTypeLabel(t, item.type)}</SelectItem>
            ))}
          </EventFilterSelect>
          <EventFilterSelect
            label={t('eventsPage.filters.severity')}
            value={severity || allValue}
            onChange={(value) => {
              setSeverity(value === allValue ? '' : value)
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allSeverities')}</SelectItem>
            {['info', 'warning', 'error'].map(value => <SelectItem key={value} value={value}>{t(`eventsPage.severities.${value}`)}</SelectItem>)}
          </EventFilterSelect>
          <EventFilterSelect
            label={t('eventsPage.filters.status')}
            value={status || allValue}
            onChange={(value) => {
              setStatus(value === allValue ? '' : value)
              resetPage()
            }}
          >
            <SelectItem value={allValue}>{t('eventsPage.filters.allStatuses')}</SelectItem>
            {['in_progress', 'succeeded', 'failed', 'canceled'].map(value => <SelectItem key={value} value={value}>{t(`eventsPage.statuses.${value}`)}</SelectItem>)}
          </EventFilterSelect>
          <label className="grid gap-1.5 text-xs text-muted-foreground">
            {t('eventsPage.filters.dateFrom')}
            <Input
              className="h-9 rounded-full"
              max={dateTo}
              type="date"
              value={dateFrom}
              onChange={(event) => {
                setDateFrom(event.target.value)
                resetPage()
              }}
            />
          </label>
          <label className="grid gap-1.5 text-xs text-muted-foreground">
            {t('eventsPage.filters.dateTo')}
            <Input
              className="h-9 rounded-full"
              min={dateFrom}
              type="date"
              value={dateTo}
              onChange={(event) => {
                setDateTo(event.target.value)
                resetPage()
              }}
            />
          </label>
        </div>
      </Card>

      <DataList
        columns={columns}
        emptyDescription={t('eventsPage.emptyDescription')}
        emptyTitle={events.isLoading ? t('common.loading') : t('eventsPage.emptyTitle')}
        items={events.data?.items ?? []}
        pagination={{
          page: events.data?.page ?? page,
          pageSize,
          total: events.data?.total ?? 0,
          totalPages: events.data?.totalPages ?? 0,
          pageInfoLabel: t('pagination.pageInfo', { page: events.data?.page ?? page, totalPages: events.data?.totalPages ?? 0, total: events.data?.total ?? 0 }),
          onPageChange: setPage,
          onPageSizeChange: (value) => {
            setPageSize(value)
            setPage(1)
          },
        }}
        rowKey={event => event.id}
        search={{
          value: search,
          placeholder: t('eventsPage.searchPlaceholder'),
          onChange: (value) => {
            setSearch(value)
            resetPage()
          },
        }}
        title={(
          <div className="flex items-center gap-2">
            <span>{t('eventsPage.listTitle')}</span>
            <Button aria-label={t('common.refresh')} disabled={events.isFetching} size="icon" variant="ghost" onClick={() => events.refetch()}>
              <RefreshCw className={`size-4 ${events.isFetching ? 'animate-spin' : ''}`} />
            </Button>
          </div>
        )}
      />

      <EventDetailSheet
        event={selectedEvent.data}
        loading={selectedEvent.isLoading}
        open={Boolean(selectedEventId)}
        onOpenChange={open => !open && setSelectedEventId('')}
      />
    </div>
  )
}

function EventFilterSelect({ children, disabled, label, onChange, value }: { children: ReactNode, disabled?: boolean, label: string, onChange: (value: string) => void, value: string }) {
  return (
    <label className="grid gap-1.5 text-xs text-muted-foreground">
      {label}
      <Select disabled={disabled} value={value} onValueChange={onChange}>
        <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
        <SelectContent>{children}</SelectContent>
      </Select>
    </label>
  )
}

function EventResource({ event }: { event: PlatformEvent }) {
  const { t } = useTranslation()
  const detail = event.detail
  const primary = detail.application?.name || detail.project?.name || event.resourceId
  const secondary = detail.deploymentTarget?.name || detail.project?.name || event.resourceType
  return (
    <div className="min-w-0">
      <p className="truncate text-sm">{primary || t('eventsPage.platformResource')}</p>
      <p className="mt-1 truncate text-xs text-muted-foreground">{secondary || event.resourceType || t('eventsPage.platformResource')}</p>
    </div>
  )
}

function EventDetailSheet({ event, loading, onOpenChange, open }: { event?: PlatformEvent, loading: boolean, onOpenChange: (open: boolean) => void, open: boolean }) {
  const { t } = useTranslation()
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-xl">
        <SheetHeader className="border-b border-border pr-10">
          <SheetTitle>{event ? eventTypeLabel(t, event.type) : t('eventsPage.detailsTitle')}</SheetTitle>
          <SheetDescription>{event?.message || (loading ? t('common.loading') : t('eventsPage.noMessage'))}</SheetDescription>
        </SheetHeader>
        {event && (
          <div className="space-y-5 px-4 pb-6">
            <div className="flex flex-wrap gap-2">
              <StatusValueBadge labelKeyPrefix="eventsPage.severities" value={event.severity} />
              <StatusValueBadge labelKeyPrefix="eventsPage.statuses" value={event.status} />
            </div>
            <DetailSection title={t('eventsPage.details.context')}>
              <DetailRow label={t('eventsPage.details.time')} value={formatAbsoluteDateTime(event.occurredAt)} />
              <DetailRow label={t('eventsPage.details.project')} value={event.detail.project?.name} />
              <DetailRow label={t('eventsPage.details.application')} value={event.detail.application?.name} />
              <DetailRow label={t('eventsPage.details.deploymentTarget')} value={event.detail.deploymentTarget?.name} />
              <DetailRow label={t('eventsPage.details.actor')} value={event.detail.actor?.name || event.detail.actor?.email} />
            </DetailSection>
            <DetailSection title={t('eventsPage.details.identifiers')}>
              <DetailRow label={t('eventsPage.details.eventId')} value={event.id} mono />
              <DetailRow label={t('eventsPage.details.resource')} value={[event.resourceType, event.resourceId].filter(Boolean).join(' / ')} mono />
              <DetailRow label={t('eventsPage.details.correlationId')} value={event.correlationId} mono />
              <DetailRow label={t('eventsPage.details.notificationDeliveries')} value={String(event.deliveryCount)} />
            </DetailSection>
            <EventSpecificDetails detail={event.detail} />
            {Object.keys(event.links).length > 0 && (
              <DetailSection title={t('eventsPage.details.links')}>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(event.links).map(([key, href]) => (
                    <a key={key} className="inline-flex h-9 items-center gap-2 rounded-full border border-border px-3 text-sm transition hover:bg-muted" href={href}>
                      {t(`eventsPage.linkNames.${key}`, { defaultValue: key })}
                      <ExternalLink className="size-3.5" />
                    </a>
                  ))}
                </div>
              </DetailSection>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}

function EventSpecificDetails({ detail }: { detail: PlatformEventSnapshot }) {
  const { t } = useTranslation()
  const context = detail.build || detail.release || detail.hook || detail.certificate || detail.gateway
  if (!context)
    return null
  const entries = Object.entries(context).filter(([, value]) => value !== '' && value !== null && value !== undefined)
  if (entries.length === 0)
    return null
  return (
    <DetailSection title={t('eventsPage.details.eventData')}>
      {entries.map(([key, value]) => (
        <DetailRow key={key} label={t(`eventsPage.detailFields.${key}`, { defaultValue: key })} value={formatDetailValue(value)} mono={key.toLowerCase().includes('id')} />
      ))}
    </DetailSection>
  )
}

function DetailSection({ children, title }: { children: ReactNode, title: string }) {
  return (
    <section className="space-y-3 border-t border-border pt-4">
      <h3 className="text-sm font-semibold">{title}</h3>
      {children}
    </section>
  )
}

function DetailRow({ label, mono, value }: { label: string, mono?: boolean, value?: string }) {
  if (!value)
    return null
  return (
    <div className="grid gap-1 sm:grid-cols-[9rem_minmax(0,1fr)] sm:gap-3">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className={`min-w-0 break-words text-sm ${mono ? 'font-mono text-xs' : ''}`}>{value}</dd>
    </div>
  )
}

function eventTypeLabel(t: ReturnType<typeof useTranslation>['t'], type: string) {
  return t(`eventsPage.types.${type.replaceAll('.', '_')}`, { defaultValue: type })
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
  return <Activity className={className} />
}

function formatDetailValue(value: unknown) {
  if (typeof value === 'string')
    return value
  if (typeof value === 'number' || typeof value === 'boolean')
    return String(value)
  return JSON.stringify(value)
}

function dateDaysAgo(days: number) {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
