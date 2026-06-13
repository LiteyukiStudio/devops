import type { ReactNode, Ref } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import type { Application, ArtifactRegistry, BuildJob, BuildRun, ClusterResource, DeploymentTarget, DeploymentTargetPayload, Environment, GatewayRoute, ProjectRuntimeConfigSet, ProjectRuntimeConfigSetPayload, Release, RepositoryBinding, RuntimeCluster } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal as XTerm } from '@xterm/xterm'
import i18next from 'i18next'
import { Activity, CalendarClock, CircleCheck, CircleX, Clock3, Download, Eye, FileCode2, Globe2, LoaderCircle, MoreHorizontal, Package, Pencil, Play, Plus, Rocket, RotateCcw, Save, ScrollText, Search, SearchCheck, Square, Terminal, Trash2, X } from 'lucide-react'
import { useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'
import { api, buildJobLogsStreamUrl, deploymentTargetDataExportUrl, releaseRuntimeTerminalUrl } from '@/api/client'
import { ApplicationBasicFields } from '@/components/common/application-basic-fields'
import { ApplicationIcon } from '@/components/common/application-icon-picker'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ContentTabs } from '@/components/common/content-tabs'
import { DataList } from '@/components/common/data-list'
import { buildRunImageRef, buildRunOptionLabel, latestDeployableBuildRuns } from '@/components/common/deployment-build-runs'
import { EditActionButton } from '@/components/common/edit-action-button'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { GatewayRouteFormFields } from '@/components/common/gateway-route-form-fields'
import { GitRepositoryPicker } from '@/components/common/git-repository-picker'
import { MotionItem, MotionList } from '@/components/common/motion'
import { PaginationController } from '@/components/common/pagination'
import { RuntimeConfigFilesEditor } from '@/components/common/runtime-config-files-editor'
import { SearchSelect } from '@/components/common/search-select'
import { SegmentedTabsList } from '@/components/common/segmented-control'
import { StatusValueBadge } from '@/components/common/status-badge'
import { TargetImageRefInput } from '@/components/common/target-image-ref-input'
import { formatElapsedDuration, formatSmartDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { WORKFLOW_STATUS_REFETCH_INTERVAL_MS } from '@/lib/polling'
import { emptyRuntimeDataVolumeRow, parseRuntimeDataVolumes, serializeRuntimeDataVolumes } from '@/lib/runtime-data-volumes'
import { APPLICATION_SLUG_MAX_LENGTH } from '@/lib/slug-limits'
import { RepositoryBindingsPage } from '@/pages/repositories/RepositoryBindingsPage'
import '@xterm/xterm/css/xterm.css'

const schema = z.object({
  name: z.string().min(1, i18next.t('apps.nameRequired')),
  slug: z.string().min(1, i18next.t('apps.slugRequired')).max(APPLICATION_SLUG_MAX_LENGTH, i18next.t('apps.slugMaxLength', { count: APPLICATION_SLUG_MAX_LENGTH })).regex(/^[a-z0-9-]+$/, i18next.t('common.lowercaseSlugOnly')),
  icon: z.string().default('box'),
  servicePort: z.coerce.number().int().min(1, i18next.t('apps.servicePortRequired')).max(65535, i18next.t('apps.servicePortMax')),
})

const repositoryBindingSchema = z.object({
  autoConfigureWebhook: z.boolean().default(true),
  cloneUrl: z.string().optional(),
  defaultBranch: z.string().optional(),
  gitAccountId: z.string().min(1, i18next.t('repositories.gitAccountRequired')),
  owner: z.string().min(1, i18next.t('repositories.ownerRequired')),
  repo: z.string().min(1, i18next.t('repositories.repoRequired')),
  webhookStatus: z.enum(['pending', 'created', 'disabled', 'failed']),
})

type ApplicationFormInput = z.input<typeof schema>
type ApplicationForm = z.output<typeof schema>
type RepositoryBindingFormInput = z.input<typeof repositoryBindingSchema>
type RepositoryBindingForm = z.output<typeof repositoryBindingSchema>
type TriggerForm = Partial<BuildRun>
type ReleaseForm = Omit<Release, 'id' | 'projectId' | 'createdBy' | 'createdAt' | 'rollbackFromId'>
type RouteForm = Omit<GatewayRoute, 'id' | 'projectId' | 'createdBy' | 'createdAt' | 'cnameName' | 'cnameTarget'>
type DeliveryConfigRow = DeploymentTarget

const triggerDefaults: TriggerForm = { applicationId: '', deploymentTargetId: '', sourceBranch: '', targetImageRef: '', targetRegistryId: '', triggerType: 'manual' }
const releaseDefaults: ReleaseForm = { applicationId: '', buildRunId: '', deploymentTargetId: '', environmentId: '', imageRef: '', message: '', revision: 1, status: 'pending', type: 'deploy' }
const routeDefaults: RouteForm = { applicationId: '', certificateStatus: 'disabled', deploymentTargetId: '', dnsStatus: 'pending', environmentId: '', host: '', isDefault: false, path: '/', servicePort: 8080, status: 'pending', tlsMode: 'http-only' }
const deploymentTargetDefaults: DeploymentTargetPayload = {
  name: '',
  environmentId: '',
  sourceType: 'repository',
  repositoryBindingId: '',
  buildProviderId: '',
  dockerfilePath: 'Dockerfile',
  buildContext: '.',
  buildDirectory: '',
  targetRegistryId: '',
  targetRepository: '',
  targetTag: 'latest',
  targetImageRef: '',
  imageRef: '',
  buildLabels: '',
  buildVariableSetIds: [],
  buildHooksEnabled: true,
  buildHookBindings: [],
  autoDeploy: false,
  branchPattern: '',
  tagPattern: '',
  concurrencyPolicy: 'queue',
  runtimeConfigSetIds: [],
  envVars: '',
  configRefs: '',
  secretRefs: '',
  configFiles: '',
  secretFiles: '',
  dataRetentionEnabled: false,
  dataCapacity: '1Gi',
  dataMountPath: '/data',
  dataVolumes: JSON.stringify([{ name: 'data', mountPath: '/data', capacity: '1Gi' }]),
  requireApproval: false,
  enabled: true,
}
const repositoryBindingDefaults: RepositoryBindingFormInput = {
  autoConfigureWebhook: true,
  cloneUrl: '',
  defaultBranch: 'main',
  gitAccountId: '',
  owner: '',
  repo: '',
  webhookStatus: 'pending',
}
const runtimeConfigDefaults: ProjectRuntimeConfigSetPayload = {
  configFiles: '',
  enabled: true,
  envVars: '',
  name: '',
  secretFiles: '',
  secretRefs: '',
}
const buildRunStatusFilters: Array<BuildRun['status']> = ['queued', 'running', 'succeeded', 'failed', 'canceled', 'lost', 'timeout']
const buildRunEventFilters: Array<BuildRun['triggerType']> = ['manual', 'push', 'tag', 'webhook', 'api', 'retry']
const APPLICATION_CONFIG_FORM_ID = 'application-config-form'

interface ApplicationPanelHandle {
  openCreateDialog: (environmentId?: string, deploymentTargetId?: string) => void
}
interface DeploymentsPanelHandle {
  openReleaseDialog: (environmentId?: string, deploymentTargetId?: string) => void
  openTargetDialog: () => void
}
interface BuildsPanelHandle {
  openTriggerDrawer: () => void
}
const buildJobProgressKeys = new Set([
  'claimed',
  'clone_repository',
  'load_dockerfile',
  'pull_image_metadata',
  'pull_base_image',
  'upload_build_context',
  'run_command',
  'export_image',
  'push_image_layers',
  'push_image_manifest',
  'registry_auth',
])

export function ApplicationConfigPage() {
  const { t } = useTranslation()
  const { projectId = '', applicationId = '' } = useParams()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState('overview')
  const shouldPollWorkflowStatus = activeTab === 'builds' || activeTab === 'deployments'
  const buildsPanelRef = useRef<BuildsPanelHandle>(null)
  const deploymentsPanelRef = useRef<DeploymentsPanelHandle>(null)
  const gatewayPanelRef = useRef<ApplicationPanelHandle>(null)
  const application = useQuery({
    queryKey: ['application', projectId, applicationId],
    queryFn: () => api.getApplication(projectId, applicationId),
    enabled: Boolean(projectId && applicationId),
  })
  const project = useQuery({ queryKey: ['project', projectId], queryFn: () => api.getProject(projectId), enabled: Boolean(projectId) })
  const repositoryBindings = useQuery({ queryKey: ['repository-bindings', projectId], queryFn: () => api.listRepositoryBindings(projectId), enabled: Boolean(projectId) })
  const registries = useQuery({ queryKey: ['registries', projectId], queryFn: () => api.listRegistries(projectId), enabled: Boolean(projectId) })
  const buildRuns = useQuery({
    queryKey: ['build-runs', projectId],
    queryFn: () => api.listBuildRuns(projectId),
    enabled: Boolean(projectId),
    refetchInterval: shouldPollWorkflowStatus ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const buildJobs = useQuery({
    queryKey: ['build-jobs', projectId],
    queryFn: () => api.listBuildJobs(projectId),
    enabled: Boolean(projectId),
    refetchInterval: shouldPollWorkflowStatus ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const environments = useQuery({ queryKey: ['environments', projectId], queryFn: () => api.listEnvironments(projectId), enabled: Boolean(projectId) })
  const releases = useQuery({
    queryKey: ['releases', projectId],
    queryFn: () => api.listReleases(projectId),
    enabled: Boolean(projectId),
    refetchInterval: activeTab === 'deployments' ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const deploymentTargets = useQuery({ queryKey: ['deployment-targets', projectId, applicationId], queryFn: () => api.listDeploymentTargets(projectId, applicationId), enabled: Boolean(projectId && applicationId) })
  const routes = useQuery({ queryKey: ['gateway-routes', projectId], queryFn: () => api.listGatewayRoutes(projectId), enabled: Boolean(projectId) })
  const deploymentTargetRows = deploymentTargets.data ?? []

  const binding = useMemo(() => (repositoryBindings.data ?? []).find(item => item.applicationId === applicationId), [applicationId, repositoryBindings.data])
  const appRepositoryBindings = useMemo(() => (repositoryBindings.data ?? []).filter(item => item.applicationId === applicationId), [applicationId, repositoryBindings.data])
  const appBuildRuns = useMemo(() => (buildRuns.data ?? []).filter(run => run.applicationId === applicationId), [applicationId, buildRuns.data])
  const releaseReadyTarget = firstReleaseReadyTarget(deploymentTargetRows, appBuildRuns)
  const appBuildRunIds = useMemo(() => new Set(appBuildRuns.map(run => run.id)), [appBuildRuns])
  const appBuildJobs = useMemo(() => (buildJobs.data ?? []).filter(job => appBuildRunIds.has(job.buildRunId)), [appBuildRunIds, buildJobs.data])
  const appReleases = useMemo(() => (releases.data ?? []).filter(release => release.applicationId === applicationId), [applicationId, releases.data])
  const appRoutes = useMemo(() => (routes.data ?? []).filter(route => route.applicationId === applicationId), [applicationId, routes.data])

  const updateForm = useForm<ApplicationFormInput, undefined, ApplicationForm>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { icon: 'box', name: '', servicePort: 8080, slug: '' },
  })

  useEffect(() => {
    if (!application.data)
      return
    updateForm.reset({
      name: application.data.name,
      slug: application.data.slug,
      icon: application.data.icon ?? 'box',
      servicePort: application.data.servicePort ?? 8080,
    })
  }, [application.data, updateForm])

  useEffect(() => {
    if (!projectId || !shouldPollWorkflowStatus)
      return
    queryClient.invalidateQueries({ queryKey: ['build-runs', projectId] })
    queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
    if (activeTab === 'deployments')
      queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
  }, [activeTab, projectId, queryClient, shouldPollWorkflowStatus])

  const updateApplication = useMutation({
    mutationFn: (payload: ApplicationForm) => api.updateApplication(projectId, applicationId, {
      name: payload.name,
      slug: payload.slug,
      icon: payload.icon,
      servicePort: payload.servicePort,
    }),
    onSuccess: (result) => {
      toast.success(t('apps.configSaved'))
      queryClient.setQueryData(['application', projectId, applicationId], result)
      queryClient.setQueryData(['applications', projectId], (items?: Application[]) => (items ?? []).map(item => item.id === result.id ? result : item))
      queryClient.invalidateQueries({ queryKey: ['applications', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  if (application.isError)
    return <ErrorState title={t('apps.loadFailedTitle')} description={t('apps.appLoadFailedDescription')} />

  return (
    <div className="grid gap-4">
      <ContentTabs
        tabs={[
          { label: t('apps.overview'), value: 'overview' },
          { label: t('apps.repoBinding'), value: 'repositories' },
          { label: t('builds'), value: 'builds' },
          { label: t('deployments'), value: 'deployments' },
          { label: t('gatewayRoutes'), value: 'gateway' },
        ]}
        tools={(
          <div className="flex items-center gap-2">
            {activeTab === 'deployments' && (
              <>
                <Button onClick={() => deploymentsPanelRef.current?.openTargetDialog()}>
                  <Plus size={16} />
                  {t('deploymentsPage.createDeploymentTarget')}
                </Button>
                <Button disabled={!releaseReadyTarget || !environments.data?.length} onClick={() => releaseReadyTarget && deploymentsPanelRef.current?.openReleaseDialog(releaseReadyTarget.environmentId, releaseReadyTarget.id)}>
                  <Package size={16} />
                  {t('deploymentsPage.createRelease')}
                </Button>
              </>
            )}
            {activeTab === 'builds' && (
              <>
                <Button disabled={!deploymentTargets.data?.some(target => target.sourceType === 'repository' && target.repositoryBindingId)} onClick={() => buildsPanelRef.current?.openTriggerDrawer()}>
                  <Play size={16} />
                  {t('buildsPage.triggerBuild')}
                </Button>
              </>
            )}
            {activeTab === 'gateway' && (
              <Button disabled={!deploymentTargets.data?.length} onClick={() => gatewayPanelRef.current?.openCreateDialog()}>
                <Globe2 size={16} />
                {t('gatewayRoutesPage.createRoute')}
              </Button>
            )}
            {activeTab === 'overview' && (
              <Button disabled={updateApplication.isPending || !updateForm.formState.isValid} form={APPLICATION_CONFIG_FORM_ID} type="submit">
                <Save size={16} />
                {t('apps.saveConfig')}
              </Button>
            )}
          </div>
        )}
        value={activeTab}
        onValueChange={setActiveTab}
      >
        <TabsContent value="overview">
          <ApplicationOverviewPanel
            app={application.data}
            buildRuns={appBuildRuns}
            deploymentTargets={deploymentTargetRows}
            releases={appReleases}
            routes={appRoutes}
          />
          <Card className="mt-4 p-4">
            <form id={APPLICATION_CONFIG_FORM_ID} onSubmit={updateForm.handleSubmit(values => updateApplication.mutate(values))}>
              <MotionList className="grid gap-4">
                <MotionItem>
                  <ApplicationBasicFields
                    compact
                    icon={updateForm.watch('icon')}
                    nameError={updateForm.formState.errors.name?.message}
                    nameField={updateForm.register('name')}
                    servicePortError={updateForm.formState.errors.servicePort?.message}
                    servicePortField={updateForm.register('servicePort', { valueAsNumber: true })}
                    slugError={updateForm.formState.errors.slug?.message}
                    slugField={updateForm.register('slug')}
                    slugMaxLength={APPLICATION_SLUG_MAX_LENGTH}
                    onIconChange={icon => updateForm.setValue('icon', icon, { shouldDirty: true, shouldValidate: true })}
                  />
                </MotionItem>
              </MotionList>
            </form>
          </Card>
        </TabsContent>
        <TabsContent value="repositories">
          <RepositoryBindingsPage
            applicationId={applicationId}
            applicationName={application.data?.name}
            embedded
            projectId={projectId}
          />
        </TabsContent>
        <TabsContent value="builds">
          <ApplicationBuildsPanel
            ref={buildsPanelRef}
            applicationId={applicationId}
            appSlug={application.data?.slug ?? ''}
            binding={binding}
            repositoryBindings={appRepositoryBindings}
            buildJobs={appBuildJobs}
            deploymentTargets={deploymentTargetRows}
            buildRuns={appBuildRuns}
            projectId={projectId}
            projectSlug={project.data?.slug ?? ''}
            registries={registries.data ?? []}
          />
        </TabsContent>
        <TabsContent value="deployments">
          <ApplicationDeploymentsPanel
            ref={deploymentsPanelRef}
            applicationId={applicationId}
            appSlug={application.data?.slug ?? ''}
            buildRuns={appBuildRuns}
            deploymentTargets={deploymentTargetRows}
            environments={environments.data ?? []}
            projectId={projectId}
            projectSlug={project.data?.slug ?? ''}
            registries={registries.data ?? []}
            repositoryBindings={appRepositoryBindings}
            releases={appReleases}
          />
        </TabsContent>
        <TabsContent value="gateway">
          <ApplicationGatewayPanel
            ref={gatewayPanelRef}
            applicationId={applicationId}
            deploymentTargets={deploymentTargetRows}
            environments={environments.data ?? []}
            projectId={projectId}
            routes={appRoutes}
            servicePort={application.data?.servicePort ?? 8080}
          />
        </TabsContent>
      </ContentTabs>
    </div>
  )
}

function ApplicationOverviewPanel({ app, buildRuns, deploymentTargets, releases, routes }: {
  app?: Application
  buildRuns: BuildRun[]
  deploymentTargets: DeploymentTarget[]
  releases: Release[]
  routes: GatewayRoute[]
}) {
  const { t } = useTranslation()
  const enabledTargets = deploymentTargets.filter(target => target.enabled).length
  const latestBuild = latestByDate(buildRuns, run => run.createdAt)
  const latestRelease = latestByDate(releases, release => release.createdAt)
  const healthyReleases = deploymentTargets.filter((target) => {
    const latest = latestReleaseForTarget(releases, target)
    return latest?.status === 'succeeded'
  }).length
  const primaryRoute = routes.find(route => route.status === 'ready') ?? routes[0]
  const recentActivities = [
    latestBuild && {
      id: `build-${latestBuild.id}`,
      label: t('apps.latestBuild'),
      meta: buildOverviewMeta(latestBuild, t),
      status: latestBuild.status,
      time: formatSmartDateTime(latestBuild.createdAt, t),
    },
    latestRelease && {
      id: `release-${latestRelease.id}`,
      label: t('apps.latestRelease'),
      meta: latestRelease.imageRef || latestRelease.id,
      status: latestRelease.status,
      time: formatReleaseTime(latestRelease, t),
    },
    primaryRoute && {
      id: `route-${primaryRoute.id}`,
      label: t('apps.primaryAccess'),
      meta: routeDisplayUrl(primaryRoute),
      status: primaryRoute.status,
      time: primaryRoute.createdAt ? formatSmartDateTime(primaryRoute.createdAt, t) : '',
    },
  ].filter(Boolean) as Array<{ id: string, label: string, meta: string, status: string, time: string }>

  return (
    <div className="grid gap-4">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <OverviewMetric
          icon={<Package className="size-4" />}
          label={t('apps.deploymentTargetHealth')}
          meta={t('apps.enabledTotal', { enabled: enabledTargets, total: deploymentTargets.length })}
          value={String(deploymentTargets.length)}
        />
        <OverviewMetric
          icon={<Activity className="size-4" />}
          label={t('apps.buildHealth')}
          meta={latestBuild ? formatSmartDateTime(latestBuild.createdAt, t) : t('apps.noRecentBuild')}
          status={latestBuild?.status}
          value={latestBuild ? t(`buildsPage.statuses.${latestBuild.status}`) : '-'}
        />
        <OverviewMetric
          icon={<Rocket className="size-4" />}
          label={t('apps.deploymentHealth')}
          meta={t('apps.deploymentReadyCount', { ready: healthyReleases, total: enabledTargets })}
          status={latestRelease?.status}
          value={latestRelease ? t(`buildsPage.statuses.${latestRelease.status}`) : '-'}
        />
        <OverviewMetric
          icon={<Globe2 className="size-4" />}
          label={t('apps.accessHealth')}
          meta={primaryRoute ? routeDisplayUrl(primaryRoute) : t('apps.noAccessRoute')}
          status={primaryRoute?.status}
          value={routes.length ? String(routes.length) : '-'}
        />
      </div>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(22rem,0.8fr)]">
        <Card className="min-w-0 p-4">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <h3 className="text-base font-semibold">{t('apps.runtimeOverview')}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{t('apps.runtimeOverviewDescription')}</p>
            </div>
            <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <ApplicationIcon name={app?.icon ?? 'box'} size={18} />
            </div>
          </div>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <OverviewItem label={t('apps.name')} value={app?.name ?? t('common.loading')} />
            <OverviewItem label={t('common.slug')} value={app?.slug ?? '-'} />
            <OverviewItem label={t('apps.buildConfigsTitle')} value={t('apps.enabledTotal', { enabled: enabledTargets, total: deploymentTargets.length })} />
          </div>
        </Card>
        <Card className="min-w-0 p-4">
          <h3 className="text-base font-semibold">{t('apps.accessEntries')}</h3>
          <div className="mt-3 grid gap-2">
            {routes.length
              ? routes.slice(0, 4).map(route => (
                  <a key={route.id} className="flex min-w-0 items-center justify-between gap-3 rounded-md border border-border px-3 py-2 text-sm transition hover:border-primary/40 hover:text-primary" href={routeDisplayUrl(route)} rel="noreferrer" target="_blank">
                    <span className="min-w-0 truncate">{routeDisplayUrl(route)}</span>
                    <StatusValueBadge labelKeyPrefix="gatewayRoutesPage.statuses" value={route.status} />
                  </a>
                ))
              : <EmptyState description={t('apps.noAccessRoute')} title={t('apps.noAccessRouteTitle')} variant="plain" />}
          </div>
        </Card>
      </div>
      <Card className="min-w-0 p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-base font-semibold">{t('apps.deploymentTargetEntries')}</h3>
            <p className="mt-1 text-sm text-muted-foreground">{t('apps.deploymentTargetEntriesDescription')}</p>
          </div>
          <Package className="size-5 shrink-0 text-muted-foreground" />
        </div>
        <div className="mt-3 grid gap-2 md:grid-cols-2">
          {deploymentTargets.length
            ? deploymentTargets.slice(0, 6).map(target => (
                <div key={target.id} className="flex min-w-0 items-center justify-between gap-3 rounded-md border border-border px-3 py-2 text-sm">
                  <span className="min-w-0 truncate" title={target.name}>{target.name}</span>
                  <div className="flex shrink-0 items-center gap-2">
                    <StatusValueBadge value={target.enabled ? 'enabled' : 'disabled'} />
                  </div>
                </div>
              ))
            : <EmptyState description={t('apps.emptyBuildConfigs')} title={t('apps.emptyBuildConfigs')} variant="plain" />}
        </div>
      </Card>
      <Card className="min-w-0 p-4">
        <h3 className="text-base font-semibold">{t('apps.recentActivity')}</h3>
        <div className="mt-3 grid gap-2">
          {recentActivities.length
            ? recentActivities.map(item => (
                <div key={item.id} className="flex min-w-0 items-center justify-between gap-3 rounded-md border border-border px-3 py-2">
                  <div className="min-w-0">
                    <div className="text-sm font-medium">{item.label}</div>
                    <div className="mt-1 truncate text-xs text-muted-foreground" title={item.meta}>{item.meta}</div>
                  </div>
                  <div className="flex shrink-0 items-center gap-3">
                    <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={item.status} />
                    {item.time && <span className="text-xs text-muted-foreground">{item.time}</span>}
                  </div>
                </div>
              ))
            : <EmptyState description={t('apps.noRecentActivityDescription')} title={t('apps.noRecentActivity')} variant="plain" />}
        </div>
      </Card>
    </div>
  )
}

function OverviewMetric({ icon, label, meta, status, value }: { icon: ReactNode, label: string, meta: string, status?: string, value: string }) {
  return (
    <Card className="min-w-0 p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">{icon}</div>
        {status && <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={status} />}
      </div>
      <div className="mt-4 text-2xl font-semibold tracking-normal">{value}</div>
      <div className="mt-1 text-sm font-medium text-foreground">{label}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground" title={meta}>{meta}</div>
    </Card>
  )
}

function OverviewItem({ icon, label, value }: { icon?: string, label: string, value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs font-medium uppercase text-muted-foreground">{label}</div>
      <div className="mt-1 flex min-w-0 items-center gap-2 text-sm text-foreground" title={value}>
        {icon && (
          <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <ApplicationIcon name={icon} size={16} />
          </span>
        )}
        <span className="truncate">{value}</span>
      </div>
    </div>
  )
}

function latestByDate<T>(items: T[], dateOf: (item: T) => string | undefined) {
  return items.reduce<T | undefined>((latest, item) => {
    if (!latest)
      return item
    return new Date(dateOf(item) ?? '').getTime() > new Date(dateOf(latest) ?? '').getTime() ? item : latest
  }, undefined)
}

function latestReleaseForTarget(releases: Release[], target: DeploymentTarget) {
  return latestByDate(
    releases.filter(release => release.deploymentTargetId === target.id),
    release => release.createdAt,
  )
}

function routeDisplayUrl(route: GatewayRoute) {
  const host = route.host.trim()
  if (!host)
    return '-'
  const protocol = route.tlsMode === 'http-only' ? 'http' : 'https'
  const path = route.path?.startsWith('/') ? route.path : `/${route.path || ''}`
  return `${protocol}://${host}${path === '/' ? '' : path}`
}

function buildOverviewMeta(run: BuildRun, t: ReturnType<typeof useTranslation>['t']) {
  const ref = run.sourceBranch || run.sourceTag || t('common.unknown')
  const image = buildRunImageRef(run) || run.targetRepository || '-'
  return `${ref} · ${image}`
}

function BuildRunFilterBar({ actor, actorOptions, branch, branchOptions, event, onActorChange, onBranchChange, onEventChange, onStatusChange, status }: {
  actor: string
  actorOptions: string[]
  branch: string
  branchOptions: string[]
  event: string
  status: string
  onActorChange: (value: string) => void
  onBranchChange: (value: string) => void
  onEventChange: (value: string) => void
  onStatusChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center gap-1 border-b border-border bg-muted/25 px-4 py-2">
      <BuildRunFilterSelect
        label={t('buildsPage.eventFilter')}
        value={event}
        onChange={onEventChange}
      >
        <option value="">{t('buildsPage.allEvents')}</option>
        {buildRunEventFilters.map(value => (
          <option key={value} value={value}>{t(`buildsPage.events.${value}`)}</option>
        ))}
      </BuildRunFilterSelect>
      <BuildRunFilterSelect
        label={t('buildsPage.statusFilter')}
        value={status}
        onChange={onStatusChange}
      >
        <option value="">{t('buildsPage.allStatuses')}</option>
        {buildRunStatusFilters.map(value => (
          <option key={value} value={value}>{t(`buildsPage.statuses.${value}`)}</option>
        ))}
      </BuildRunFilterSelect>
      <BuildRunFilterSelect
        label={t('buildsPage.branchFilter')}
        value={branch}
        onChange={onBranchChange}
      >
        <option value="">{t('buildsPage.allBranches')}</option>
        {branchOptions.map(value => (
          <option key={value} value={value}>{value}</option>
        ))}
      </BuildRunFilterSelect>
      <BuildRunFilterSelect
        label={t('buildsPage.actorFilter')}
        value={actor}
        onChange={onActorChange}
      >
        <option value="">{t('buildsPage.allActors')}</option>
        {actorOptions.map(value => (
          <option key={value} value={value}>{shortActorLabel(value)}</option>
        ))}
      </BuildRunFilterSelect>
    </div>
  )
}

function BuildRunFilterSelect({ children, label, onChange, value }: {
  children: ReactNode
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <label className="min-w-32">
      <span className="sr-only">{label}</span>
      <Select
        aria-label={label}
        className="h-8 rounded-md border-transparent bg-transparent px-2.5 pr-8 text-sm text-muted-foreground shadow-none hover:bg-background/70 focus-visible:border-primary/40 focus-visible:ring-primary/20"
        value={value}
        onChange={event => onChange(event.target.value)}
      >
        {children}
      </Select>
    </label>
  )
}

function ApplicationBuildsPanel({ applicationId, appSlug, binding, deploymentTargets, buildJobs, buildRuns, projectId, projectSlug, ref, registries, repositoryBindings }: {
  applicationId: string
  appSlug: string
  binding?: { defaultBranch: string, gitAccountId: string, owner: string, repo: string }
  repositoryBindings: RepositoryBinding[]
  deploymentTargets: DeliveryConfigRow[]
  buildJobs: BuildJob[]
  buildRuns: BuildRun[]
  projectId: string
  projectSlug: string
  ref?: Ref<BuildsPanelHandle>
  registries: ArtifactRegistry[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [branchSearch, setBranchSearch] = useState('')
  const [runSearch, setRunSearch] = useState('')
  const [runsPage, setRunsPage] = useState(1)
  const [runsPageSize, setRunsPageSize] = useState(10)
  const [eventFilter, setEventFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [branchFilter, setBranchFilter] = useState('')
  const [actorFilter, setActorFilter] = useState('')
  const [logJob, setLogJob] = useState<BuildJob | null>(null)
  const [logContent, setLogContent] = useState('')
  const [logStreaming, setLogStreaming] = useState(false)
  const form = useForm<TriggerForm>({ defaultValues: triggerDefaults, mode: 'onChange' })
  const selectedDeploymentTarget = deploymentTargets.find(config => config.id === form.watch('deploymentTargetId')) ?? firstSelectableDeploymentTarget(deploymentTargets)
  const selectedBinding = repositoryBindings.find(item => item.id === selectedDeploymentTarget?.repositoryBindingId) ?? binding
  const selectedRegistry = registries.find(registry => registry.id === form.watch('targetRegistryId'))
  const targetImagePrefix = selectedRegistry ? registryInputPrefix(selectedRegistry) : ''
  const buildJobMap = useMemo(() => {
    const grouped = new Map<string, BuildJob[]>()
    for (const job of buildJobs) {
      const jobs = grouped.get(job.buildRunId) ?? []
      jobs.push(job)
      grouped.set(job.buildRunId, jobs)
    }
    for (const jobs of grouped.values()) {
      jobs.sort((left, right) => new Date(right.createdAt ?? '').getTime() - new Date(left.createdAt ?? '').getTime())
    }
    return grouped
  }, [buildJobs])
  const buildRunsPage = useQuery({
    queryKey: ['build-runs-page', projectId, applicationId, runsPage, runsPageSize, runSearch, eventFilter, statusFilter, branchFilter, actorFilter],
    queryFn: () => api.listBuildRunsPage(projectId, {
      applicationId,
      createdBy: actorFilter || undefined,
      page: runsPage,
      pageSize: runsPageSize,
      search: runSearch.trim() || undefined,
      sortBy: 'createdAt',
      sortOrder: 'desc',
      sourceBranch: branchFilter || undefined,
      status: statusFilter ? statusFilter as BuildRun['status'] : undefined,
      triggerType: eventFilter ? eventFilter as BuildRun['triggerType'] : undefined,
    }),
    enabled: Boolean(projectId && applicationId),
    refetchInterval: projectId && applicationId ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const pagedRuns = buildRunsPage.data?.items ?? []
  const runsTotal = buildRunsPage.data?.total ?? 0
  const branchFilterOptions = useMemo(() => uniqueBuildFilterValues([
    selectedBinding?.defaultBranch,
    ...buildRuns.map(run => run.sourceBranch),
    branchFilter,
  ]), [selectedBinding?.defaultBranch, branchFilter, buildRuns])
  const actorFilterOptions = useMemo(() => uniqueBuildFilterValues([
    ...buildRuns.map(run => run.createdBy),
    actorFilter,
  ]), [actorFilter, buildRuns])
  const updateRunSearch = (value: string) => {
    setRunSearch(value)
    setRunsPage(1)
  }
  const updateEventFilter = (value: string) => {
    setEventFilter(value)
    setRunsPage(1)
  }
  const updateStatusFilter = (value: string) => {
    setStatusFilter(value)
    setRunsPage(1)
  }
  const updateBranchFilter = (value: string) => {
    setBranchFilter(value)
    setRunsPage(1)
  }
  const updateActorFilter = (value: string) => {
    setActorFilter(value)
    setRunsPage(1)
  }
  const branches = useQuery({
    queryKey: ['git-branches', selectedBinding?.gitAccountId, selectedBinding?.owner, selectedBinding?.repo, branchSearch],
    queryFn: () => api.listGitBranches(selectedBinding?.gitAccountId ?? '', selectedBinding?.owner ?? '', selectedBinding?.repo ?? '', { search: branchSearch, limit: 50 }),
    enabled: Boolean(selectedBinding),
  })
  const triggerBuild = useMutation({
    mutationFn: (values: TriggerForm) => api.triggerBuildRun(projectId, { ...values, applicationId }),
    onSuccess: () => {
      toast.success(t('buildsPage.buildQueued'))
      setDialogOpen(false)
      form.reset(triggerDefaults)
      queryClient.invalidateQueries({ queryKey: ['build-runs', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-runs-page', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
      queryClient.invalidateQueries({ queryKey: ['application', projectId, applicationId] })
      queryClient.invalidateQueries({ queryKey: ['applications', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const retryBuild = useMutation({
    mutationFn: (runId: string) => api.retryBuildRun(projectId, runId),
    onSuccess: () => {
      toast.success(t('buildsPage.retryQueued'))
      queryClient.invalidateQueries({ queryKey: ['build-runs', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-runs-page', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const cancelBuild = useMutation({
    mutationFn: (runId: string) => api.cancelBuildRun(projectId, runId),
    onSuccess: () => {
      toast.success(t('buildsPage.cancelled'))
      queryClient.invalidateQueries({ queryKey: ['build-runs', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-runs-page', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteBuild = useMutation({
    mutationFn: (runId: string) => api.deleteBuildRun(projectId, runId),
    onSuccess: () => {
      toast.success(t('buildsPage.deleted'))
      queryClient.invalidateQueries({ queryKey: ['build-runs', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-runs-page', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  useImperativeHandle(ref, () => ({
    openTriggerDrawer: () => setDialogOpen(true),
  }))
  useEffect(() => {
    const logJobId = logJob?.id
    if (!logJobId) {
      setLogContent('')
      setLogStreaming(false)
      return
    }
    setLogContent('')
    setLogStreaming(true)
    const stream = new EventSource(buildJobLogsStreamUrl(projectId, logJobId, 0), { withCredentials: true })
    const handleChunk = (event: Event) => {
      try {
        const payload = JSON.parse((event as MessageEvent).data) as { content?: string }
        if (payload.content)
          setLogContent(current => current + payload.content)
      }
      catch {
      }
    }
    const handleDone = () => {
      setLogStreaming(false)
      stream.close()
      queryClient.invalidateQueries({ queryKey: ['build-runs-page', projectId] })
      queryClient.invalidateQueries({ queryKey: ['build-jobs', projectId] })
    }
    stream.addEventListener('chunk', handleChunk)
    stream.addEventListener('done', handleDone)
    stream.onerror = () => {
      setLogStreaming(false)
      stream.close()
    }
    return () => {
      stream.removeEventListener('chunk', handleChunk)
      stream.removeEventListener('done', handleDone)
      stream.close()
    }
  }, [logJob, projectId, queryClient])

  useEffect(() => {
    if (!dialogOpen)
      return
    const defaultConfig = firstSelectableDeploymentTarget(deploymentTargets)
    form.reset({
      ...triggerDefaults,
      applicationId,
      deploymentTargetId: defaultConfig?.id ?? '',
      sourceBranch: (repositoryBindings.find(item => item.id === defaultConfig?.repositoryBindingId) ?? binding)?.defaultBranch || 'main',
      targetImageRef: deploymentTargetImageRef(defaultConfig) || defaultTargetImageRef(undefined, projectSlug, appSlug),
      targetRegistryId: defaultConfig?.targetRegistryId ?? '',
    })
  }, [applicationId, appSlug, binding, deploymentTargets, dialogOpen, form, projectSlug, repositoryBindings])

  useEffect(() => {
    if (!dialogOpen || !selectedDeploymentTarget)
      return
    const nextBinding = repositoryBindings.find(item => item.id === selectedDeploymentTarget.repositoryBindingId) ?? binding
    if (nextBinding)
      form.setValue('sourceBranch', nextBinding.defaultBranch || 'main', { shouldDirty: true, shouldValidate: true })
    if (selectedDeploymentTarget.targetRegistryId)
      form.setValue('targetRegistryId', selectedDeploymentTarget.targetRegistryId, { shouldDirty: true, shouldValidate: true })
    const configTargetImageRef = deploymentTargetImageRef(selectedDeploymentTarget)
    if (configTargetImageRef)
      form.setValue('targetImageRef', configTargetImageRef, { shouldDirty: true, shouldValidate: true })
  }, [binding, dialogOpen, form, repositoryBindings, selectedDeploymentTarget])

  useEffect(() => {
    if (!dialogOpen || !registries.length || form.getValues('targetRegistryId'))
      return
    const defaultRegistry = registries.find(registry => registry.credentialSet && registry.isDefault) ?? registries.find(registry => registry.credentialSet) ?? registries.find(registry => registry.isDefault) ?? registries[0]
    form.setValue('targetRegistryId', defaultRegistry.id, { shouldDirty: true, shouldValidate: true })
    if (!form.getValues('targetImageRef')) {
      form.setValue('targetImageRef', defaultTargetImageRef(defaultRegistry, projectSlug, appSlug), { shouldDirty: true, shouldValidate: true })
    }
  }, [appSlug, dialogOpen, form, projectSlug, registries])

  return (
    <div className="grid gap-4">
      {repositoryBindings.length || binding
        ? (
            <div className="overflow-hidden rounded-lg border border-border bg-card">
              <div className="flex flex-col gap-3 border-b border-border bg-muted/45 px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0">
                  <h2 className="text-base font-semibold">{t('buildsPage.workflowRunCount', { count: runsTotal })}</h2>
                  <p className="mt-1 text-sm text-muted-foreground">{t('buildsPage.applicationRunsDescription')}</p>
                </div>
                <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
                  <div className="relative min-w-0 sm:w-80">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      className="h-9 pl-9"
                      placeholder={t('buildsPage.filterRuns')}
                      value={runSearch}
                      onChange={event => updateRunSearch(event.target.value)}
                    />
                  </div>
                </div>
              </div>
              <BuildRunFilterBar
                actor={actorFilter}
                actorOptions={actorFilterOptions}
                branch={branchFilter}
                branchOptions={branchFilterOptions}
                event={eventFilter}
                status={statusFilter}
                onActorChange={updateActorFilter}
                onBranchChange={updateBranchFilter}
                onEventChange={updateEventFilter}
                onStatusChange={updateStatusFilter}
              />
              {pagedRuns.length
                ? (
                    <div className="divide-y divide-border">
                      {pagedRuns.map((run) => {
                        const jobs = buildJobMap.get(run.id) ?? []
                        const latestJob = jobs[0]
                        const config = deploymentTargets.find(config => config.id === run.deploymentTargetId)
                        const rowBinding = repositoryBindings.find(binding => binding.id === config?.repositoryBindingId) ?? binding
                        if (!rowBinding)
                          return null
                        return (
                          <BuildRunRow
                            key={run.id}
                            binding={rowBinding}
                            deploymentTargetName={config?.name}
                            jobs={jobs}
                            latestJob={latestJob}
                            run={run}
                            canceling={cancelBuild.isPending}
                            deleting={deleteBuild.isPending}
                            retrying={retryBuild.isPending}
                            onCancel={() => cancelBuild.mutate(run.id)}
                            onDelete={() => deleteBuild.mutate(run.id)}
                            onOpenLogs={job => setLogJob(job)}
                            onRetry={() => retryBuild.mutate(run.id)}
                          />
                        )
                      })}
                    </div>
                  )
                : <EmptyState title={t('buildsPage.emptyRuns')} variant="plain" />}
              <div className="border-t border-border px-4 py-4">
                <PaginationController
                  initialPage={runsPage}
                  pageSize={runsPageSize}
                  pageSizeOptions={[10, 20, 50]}
                  total={runsTotal}
                  onPageChange={setRunsPage}
                  onPageSizeChange={(pageSize) => {
                    setRunsPageSize(pageSize)
                    setRunsPage(1)
                  }}
                />
              </div>
            </div>
          )
        : <EmptyState title={t('buildsPage.repositoryBindingRequired')} />}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('buildsPage.triggerBuild')}</DialogTitle>
            <DialogDescription>{t('buildsPage.triggerDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={form.handleSubmit(values => triggerBuild.mutate(values))}>
            <Field hint={t('buildsPage.buildConfigHint')} label={t('buildsPage.buildConfig')} required>
              <Select {...form.register('deploymentTargetId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {deploymentTargets.map(config => <option key={config.id} value={config.id}>{config.name}</option>)}
              </Select>
            </Field>
            <Field label={t('repositories.defaultBranch')} required>
              <SearchSelect
                disabled={!selectedBinding}
                emptyLabel={selectedBinding ? t('common.noOptions') : t('buildsPage.repositoryBindingRequired')}
                limited={branches.data?.limited}
                loading={branches.isFetching}
                options={branchOptions(branches.data?.items ?? [], form.watch('sourceBranch'))}
                placeholder={t('repositories.defaultBranchPlaceholder')}
                search={branchSearch}
                value={form.watch('sourceBranch') || ''}
                onSearchChange={setBranchSearch}
                onValueChange={value => form.setValue('sourceBranch', value, { shouldDirty: true, shouldValidate: true })}
              />
            </Field>
            <Field label={t('buildsPage.targetRegistry')} required>
              <Select {...form.register('targetRegistryId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {registries.map(registry => <option key={registry.id} value={registry.id}>{registryOptionLabel(registry)}</option>)}
              </Select>
            </Field>
            <Field hint={t('buildsPage.targetImageRefHint')} label={t('buildsPage.targetImageRef')} required>
              <TargetImageRefInput
                placeholder={t('buildsPage.targetImageRefPlaceholder')}
                prefix={targetImagePrefix}
                register={form.register('targetImageRef', { required: true })}
              />
            </Field>
            <Field hint={t('buildsPage.inheritedModuleHint')} label={t('buildsPage.dockerfilePath')}>
              <Input readOnly value={selectedDeploymentTarget?.dockerfilePath || 'Dockerfile'} />
            </Field>
            <Field hint={t('buildsPage.inheritedModuleHint')} label={t('buildsPage.buildContext')}>
              <Input readOnly value={selectedDeploymentTarget?.buildContext || '.'} />
            </Field>
            <DialogFooter><Button disabled={!form.formState.isValid || triggerBuild.isPending} type="submit">{t('buildsPage.queueBuild')}</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <BuildLogPanel
        job={logJob}
        content={logContent}
        loading={logStreaming}
        onClose={() => setLogJob(null)}
      />
    </div>
  )
}

function BuildRunRow({ binding, deploymentTargetName, canceling, deleting, jobs, latestJob, onCancel, onDelete, onOpenLogs, onRetry, retrying, run }: {
  binding: { cloneUrl?: string, defaultBranch: string, gitAccountId: string, owner: string, repo: string }
  deploymentTargetName?: string
  canceling: boolean
  deleting: boolean
  jobs: BuildJob[]
  latestJob?: BuildJob
  onCancel: () => void
  onDelete: () => void
  onOpenLogs: (job: BuildJob) => void
  onRetry: () => void
  retrying: boolean
  run: BuildRun
}) {
  const { t } = useTranslation()
  const branch = run.sourceBranch || run.sourceTag || binding.defaultBranch || 'main'
  const targetImage = buildRunImageRef(run)
  const commit = shortCommit(run.sourceCommit)
  const triggerActor = buildRunTriggerActor(run)
  const sourceAuthor = buildRunSourceAuthor(run)
  const imageReady = run.status === 'succeeded' && Boolean(targetImage)
  const liveState = buildRunLiveState(run, latestJob, t)
  const duration = formatBuildDuration(run, t)
  const commitUrl = buildCommitUrl(binding, run.sourceCommit)
  const authorUrl = buildAuthorUrl(binding, run)
  const canCancel = run.status === 'queued' || run.status === 'running'
  const canDelete = ['succeeded', 'failed', 'canceled', 'lost', 'timeout'].includes(run.status)
  const copyImageRef = () => {
    if (!targetImage)
      return
    navigator.clipboard.writeText(targetImage)
      .then(() => toast.success(t('buildsPage.imageRefCopied')))
      .catch(error => toast.error(error.message))
  }
  return (
    <div className="grid gap-2 px-4 py-3 transition-colors hover:bg-muted/35 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
      <div className="flex min-w-0 gap-2.5">
        <BuildRunStatusIcon status={run.status} />
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1">
            <h3 className="truncate text-sm font-semibold text-foreground" title={buildRunTitle(run, t, deploymentTargetName)}>
              {buildRunTitle(run, t, deploymentTargetName)}
            </h3>
            <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={run.status} />
          </div>
          <div className="mt-1.5 flex min-w-0 flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
            <span className="inline-flex max-w-full min-w-0 items-center gap-1.5 rounded-md border border-border bg-muted/60 px-2 py-1">
              <span className="shrink-0 font-mono font-medium text-primary">{branch}</span>
              {commitUrl
                ? (
                    <a className="min-w-0 truncate font-medium text-foreground/80 transition-colors hover:text-primary" href={commitUrl} rel="noreferrer" target="_blank" title={`${binding.owner}/${binding.repo}`}>
                      {binding.owner}
                      /
                      {binding.repo}
                    </a>
                  )
                : (
                    <span className="min-w-0 truncate font-medium text-foreground/80" title={`${binding.owner}/${binding.repo}`}>
                      {binding.owner}
                      /
                      {binding.repo}
                    </span>
                  )}
              <span className="shrink-0 font-mono text-muted-foreground">
                #
                {shortBuildId(run.id)}
              </span>
              <span className="shrink-0">{t('buildsPage.triggeredBy', { actor: triggerActor })}</span>
              {commit && (
                <>
                  {sourceAuthor && <span className="shrink-0">·</span>}
                  {sourceAuthor && (
                    authorUrl
                      ? <a className="shrink-0 transition-colors hover:text-primary" href={authorUrl} rel="noreferrer" target="_blank">{t('buildsPage.committedBy', { actor: sourceAuthor })}</a>
                      : <span className="shrink-0">{t('buildsPage.committedBy', { actor: sourceAuthor })}</span>
                  )}
                  <span className="shrink-0">{t('buildsPage.commitAction')}</span>
                  {commitUrl
                    ? <a className="shrink-0 font-mono text-foreground/70 transition-colors hover:text-primary" href={commitUrl} rel="noreferrer" target="_blank">{commit}</a>
                    : <span className="shrink-0 font-mono text-foreground/70">{commit}</span>}
                </>
              )}
            </span>
            <button
              className="inline-flex max-w-full min-w-0 items-center gap-1.5 rounded-md border border-border bg-muted/60 px-2 py-1 text-left transition-colors hover:border-primary/50 hover:text-primary disabled:hover:border-border disabled:hover:text-muted-foreground"
              disabled={!imageReady}
              title={imageReady ? targetImage : liveState}
              type="button"
              onClick={copyImageRef}
            >
              {imageReady
                ? <Package className="size-3.5 shrink-0 text-muted-foreground" />
                : <BuildRunStatusIcon compact status={run.status} />}
              <span className="min-w-0 truncate font-mono">
                {imageReady ? targetImage : liveState}
              </span>
            </button>
          </div>
        </div>
      </div>
      <div className="flex min-w-0 items-start justify-between gap-2 lg:min-w-72">
        <div className="grid min-w-0 gap-1 text-sm text-muted-foreground lg:justify-items-start">
          <span className="inline-flex min-w-0 items-center gap-2">
            <CalendarClock className="size-4 shrink-0" />
            <span className="truncate">
              {formatBuildDate(run, t)}
              {duration && (
                <>
                  {' '}
                  ·
                  {' '}
                  {duration}
                </>
              )}
            </span>
          </span>
          <span className="inline-flex min-w-0 items-center gap-2">
            <Clock3 className="size-4 shrink-0" />
            <span className="truncate">
              {latestJob
                ? t('buildsPage.latestJobSummary', { attempts: latestJob.attempts, id: shortBuildId(latestJob.id) })
                : t('buildsPage.noBuildJob')}
            </span>
          </span>
          {jobs.length > 1 && <span className="text-xs">{t('buildsPage.jobCount', { count: jobs.length })}</span>}
        </div>
        <Popover>
          <PopoverTrigger asChild>
            <Button aria-label={t('buildsPage.runActions')} className="shrink-0" size="icon" variant="ghost">
              <MoreHorizontal className="size-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-64 max-w-[calc(100vw-2rem)] p-1">
            <Button className="h-auto w-full justify-start gap-2 whitespace-normal text-left" disabled={retrying} variant="ghost" onClick={onRetry}>
              <RotateCcw className="size-4 shrink-0" />
              <span className="min-w-0">{t('buildsPage.retry')}</span>
            </Button>
            {canCancel && (
              <ConfirmDialog
                confirmText={t('buildsPage.cancelBuildConfirm')}
                description={t('buildsPage.cancelBuildDescription')}
                pending={canceling}
                title={t('buildsPage.cancelBuildTitle')}
                onConfirm={onCancel}
              >
                <Button className="h-auto w-full justify-start gap-2 whitespace-normal text-left text-danger hover:text-danger" disabled={canceling} variant="ghost">
                  <Square className="size-4 shrink-0" />
                  <span className="min-w-0">{t('buildsPage.cancelBuild')}</span>
                </Button>
              </ConfirmDialog>
            )}
            {canDelete && (
              <ConfirmDialog
                confirmText={t('common.delete')}
                description={t('buildsPage.deleteBuildDescription')}
                pending={deleting}
                title={t('buildsPage.deleteBuildTitle')}
                onConfirm={onDelete}
              >
                <Button className="h-auto w-full justify-start gap-2 whitespace-normal text-left text-danger hover:text-danger" disabled={deleting} variant="ghost">
                  <Trash2 className="size-4 shrink-0" />
                  <span className="min-w-0">{t('buildsPage.deleteBuild')}</span>
                </Button>
              </ConfirmDialog>
            )}
            <Button className="h-auto w-full justify-start gap-2 whitespace-normal text-left" disabled={!latestJob} variant="ghost" onClick={() => latestJob && onOpenLogs(latestJob)}>
              <ScrollText className="size-4 shrink-0" />
              <span className="min-w-0">{t('buildsPage.viewLogsStream')}</span>
            </Button>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  )
}

function BuildLogPanel({ content, job, loading, onClose }: {
  content: string
  job: BuildJob | null
  loading: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  if (!job)
    return null
  return (
    <div className="fixed inset-0 z-50 bg-black/20" onClick={onClose}>
      <aside
        className="absolute right-0 top-0 flex h-full w-full max-w-3xl flex-col border-l border-border bg-background shadow-xl"
        onClick={event => event.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <h2 className="truncate text-base font-semibold">{t('buildsPage.logsTitle', { id: shortBuildId(job.id) })}</h2>
              <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={job.status} />
            </div>
            <p className="text-sm text-muted-foreground">{loading ? t('buildsPage.logsStreaming') : t('buildsPage.logsUpdated')}</p>
          </div>
          <Button aria-label={t('common.close')} size="icon" variant="ghost" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>
        <pre className="min-h-0 flex-1 overflow-auto bg-zinc-950 p-4 font-mono text-sm leading-6 text-zinc-100">
          {content || t('buildsPage.noLogs')}
        </pre>
      </aside>
    </div>
  )
}

function BuildRunStatusIcon({ compact = false, status }: { compact?: boolean, status: string }) {
  const className = compact ? 'size-3.5 shrink-0' : 'mt-0.5 size-5 shrink-0'
  if (status === 'succeeded')
    return <CircleCheck className={`${className} text-emerald-600`} />
  if (status === 'failed' || status === 'lost' || status === 'timeout')
    return <CircleX className={`${className} text-rose-600`} />
  if (status === 'running')
    return <LoaderCircle className={`${className} animate-spin text-primary`} />
  return <Clock3 className={`${className} text-muted-foreground`} />
}

function buildRunTitle(run: BuildRun, t: ReturnType<typeof useTranslation>['t'], deploymentTargetName?: string) {
  let title = t('buildsPage.runTitleManual')
  if (run.triggerType === 'webhook' || run.triggerType === 'push')
    title = t('buildsPage.runTitlePush')
  else if (run.triggerType === 'tag')
    title = t('buildsPage.runTitleTag')
  else if (run.triggerType === 'api')
    title = t('buildsPage.runTitleApi')
  else if (run.triggerType === 'retry')
    title = t('buildsPage.runTitleRetry')
  return deploymentTargetName ? t('buildsPage.runTitleWithConfig', { config: deploymentTargetName, title }) : title
}

function shortCommit(value: string) {
  return value ? value.slice(0, 7) : ''
}

function shortBuildId(value: string) {
  const index = value.indexOf('_')
  if (index >= 0)
    return value.slice(index + 1, index + 9)
  return value.slice(0, 8)
}

function shortActorLabel(value: string) {
  if (!value)
    return '-'
  const index = value.indexOf('_')
  if (index >= 0)
    return value.slice(index + 1, index + 9)
  return value.length > 12 ? value.slice(0, 12) : value
}

function buildRunTriggerActor(run: BuildRun) {
  return run.triggeredByName || run.triggeredByEmail || shortActorLabel(run.createdBy)
}

function buildRunSourceAuthor(run: BuildRun) {
  return run.sourceAuthorName || run.sourceAuthorEmail
}

function buildCommitUrl(binding: { cloneUrl?: string, owner: string, repo: string }, commit: string) {
  if (!commit)
    return ''
  const repositoryUrl = repositoryBrowserUrl(binding)
  return repositoryUrl ? `${repositoryUrl}/commit/${encodeURIComponent(commit)}` : ''
}

function buildAuthorUrl(binding: { cloneUrl?: string }, run: BuildRun) {
  const username = gitAuthorUsername(run)
  if (!username)
    return ''
  const host = repositoryHostUrl(binding)
  return host ? `${host}/${encodeURIComponent(username)}` : ''
}

function gitAuthorUsername(run: BuildRun) {
  const name = run.sourceAuthorName?.trim()
  if (name && /^[\w.-]+$/.test(name))
    return name
  const emailPrefix = run.sourceAuthorEmail?.split('@')[0]?.trim()
  if (emailPrefix && /^[\w.-]+$/.test(emailPrefix))
    return emailPrefix
  return ''
}

function repositoryBrowserUrl(binding: { cloneUrl?: string, owner: string, repo: string }) {
  const host = repositoryHostUrl(binding)
  if (!host)
    return ''
  return `${host}/${encodeURIComponent(binding.owner)}/${encodeURIComponent(binding.repo)}`
}

function repositoryHostUrl(binding: { cloneUrl?: string }) {
  const cloneUrl = binding.cloneUrl?.trim()
  if (!cloneUrl)
    return ''
  const httpsUrl = cloneUrl
    .replace(/^git@([^:]+):(.+)$/, 'https://$1/$2')
    .replace(/\.git$/, '')
  try {
    const url = new URL(httpsUrl)
    return `${url.protocol}//${url.host}`
  }
  catch {
    return ''
  }
}

function buildRunLiveState(run: BuildRun, latestJob: BuildJob | undefined, t: ReturnType<typeof useTranslation>['t']) {
  const progress = buildJobProgressLabel(latestJob?.message, t)
  if (latestJob?.status === 'running' && progress)
    return progress
  return t(`buildsPage.statuses.${run.status}`)
}

function buildJobProgressLabel(message: string | undefined, t: ReturnType<typeof useTranslation>['t']) {
  const key = message?.trim()
  if (!key || !buildJobProgressKeys.has(key))
    return ''
  return t(`buildsPage.progress.${key}`)
}

function uniqueBuildFilterValues(values: Array<string | undefined>) {
  return [...new Set(values.map(value => value?.trim()).filter((value): value is string => Boolean(value)))]
    .sort((left, right) => left.localeCompare(right))
}

function formatBuildDate(run: BuildRun, t: ReturnType<typeof useTranslation>['t']) {
  return formatSmartDateTime(run.createdAt, t)
}

function formatBuildDuration(run: BuildRun, t: ReturnType<typeof useTranslation>['t']) {
  return formatElapsedDuration(run.startedAt, run.finishedAt, run.status === 'running', t)
}

function ApplicationDeploymentsPanel({ applicationId, appSlug, buildRuns, deploymentTargets, environments, projectId, projectSlug, ref, registries, releases, repositoryBindings }: {
  applicationId: string
  appSlug: string
  buildRuns: BuildRun[]
  deploymentTargets: DeploymentTarget[]
  environments: Environment[]
  projectId: string
  projectSlug: string
  ref?: React.Ref<DeploymentsPanelHandle>
  registries: ArtifactRegistry[]
  repositoryBindings: RepositoryBinding[]
  releases: Release[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [targetDialogOpen, setTargetDialogOpen] = useState(false)
  const [editingTarget, setEditingTarget] = useState<DeploymentTarget | null>(null)
  const [targetConfigFilesValid, setTargetConfigFilesValid] = useState(true)
  const [targetSecretFilesValid, setTargetSecretFilesValid] = useState(true)
  const [logRelease, setLogRelease] = useState<Release | null>(null)
  const [logView, setLogView] = useState<'deployment' | 'runtime'>('deployment')
  const [consoleRelease, setConsoleRelease] = useState<Release | null>(null)
  const [consoleContainer, setConsoleContainer] = useState('')
  const [targetToDelete, setTargetToDelete] = useState<DeploymentTarget | null>(null)
  const [runtimeConfigDialogOpen, setRuntimeConfigDialogOpen] = useState(false)
  const [editingRuntimeConfigSet, setEditingRuntimeConfigSet] = useState<ProjectRuntimeConfigSet | null>(null)
  const [runtimeConfigFilesValid, setRuntimeConfigFilesValid] = useState(true)
  const [runtimeSecretFilesValid, setRuntimeSecretFilesValid] = useState(true)
  const [runtimeConfigRestartSetId, setRuntimeConfigRestartSetId] = useState('')
  const [runtimeConfigRestartAffectedCount, setRuntimeConfigRestartAffectedCount] = useState(0)
  const [repositoryBindingDialogOpen, setRepositoryBindingDialogOpen] = useState(false)
  const [repositoryBranchSearch, setRepositoryBranchSearch] = useState('')
  const form = useForm<ReleaseForm>({ defaultValues: releaseDefaults, mode: 'onChange' })
  const targetForm = useForm<DeploymentTargetPayload>({ defaultValues: deploymentTargetDefaults, mode: 'onChange' })
  const runtimeConfigForm = useForm<ProjectRuntimeConfigSetPayload>({ defaultValues: runtimeConfigDefaults, mode: 'onChange' })
  const repositoryBindingForm = useForm<RepositoryBindingFormInput, undefined, RepositoryBindingForm>({
    defaultValues: repositoryBindingDefaults,
    mode: 'onChange',
    resolver: zodResolver(repositoryBindingSchema),
  })
  const environmentMap = useMemo(() => Object.fromEntries(environments.map(environment => [environment.id, environment])), [environments])
  const buildRunMap = useMemo(() => Object.fromEntries(buildRuns.map(run => [run.id, run])), [buildRuns])
  const latestReleaseByTarget = useMemo(() => {
    const output: Record<string, Release> = {}
    for (const release of releases) {
      const key = deploymentReleaseKey(release.environmentId, release.deploymentTargetId)
      const existing = output[key]
      if (!existing || new Date(release.createdAt).getTime() > new Date(existing.createdAt).getTime())
        output[key] = release
    }
    return output
  }, [releases])
  const deployableBuildRuns = useMemo(() => latestDeployableBuildRuns(buildRuns), [buildRuns])
  const selectedDeploymentTargetId = form.watch('deploymentTargetId')
  const selectedReleaseTarget = deploymentTargets.find(target => target.id === selectedDeploymentTargetId)
  const selectableBuildRuns = useMemo(
    () => selectedDeploymentTargetId ? deployableBuildRuns.filter(run => run.deploymentTargetId === selectedDeploymentTargetId) : deployableBuildRuns,
    [deployableBuildRuns, selectedDeploymentTargetId],
  )
  const targetSourceType = targetForm.watch('sourceType')
  const targetRepositoryBindingId = targetForm.watch('repositoryBindingId')
  const targetDataRetentionEnabled = normalizeBoolean(targetForm.watch('dataRetentionEnabled'), false)
  const targetDataVolumesValue = targetForm.watch('dataVolumes')
  const targetDataVolumes = useMemo(
    () => parseRuntimeDataVolumes(targetDataVolumesValue, targetForm.getValues('dataMountPath') || '/data', targetForm.getValues('dataCapacity') || '1Gi'),
    [targetDataVolumesValue, targetForm],
  )
  const watchedTargetValues = targetForm.watch()
  const selectedRuntimeConfigSetIds = normalizeStringIds(targetForm.watch('runtimeConfigSetIds'))
  const selectedTargetRepositoryBinding = repositoryBindings.find(binding => binding.id === targetRepositoryBindingId)
  const targetRegistry = registries.find(registry => registry.id === targetForm.watch('targetRegistryId'))
  const targetImagePrefix = targetRegistry ? registryInputPrefix(targetRegistry) : ''
  const gitProviders = useQuery({ queryKey: ['git-providers'], queryFn: () => api.listGitProviders(), enabled: repositoryBindingDialogOpen })
  const gitAccounts = useQuery({ queryKey: ['git-accounts'], queryFn: () => api.listGitAccounts(), enabled: repositoryBindingDialogOpen })
  const selectedRepositoryAccountId = repositoryBindingForm.watch('gitAccountId')
  const selectedRepositoryOwner = repositoryBindingForm.watch('owner')
  const selectedRepositoryName = repositoryBindingForm.watch('repo')
  const repositoryBranches = useQuery({
    queryKey: ['git-branches', selectedRepositoryAccountId, selectedRepositoryOwner, selectedRepositoryName, repositoryBranchSearch],
    queryFn: () => api.listGitBranches(selectedRepositoryAccountId || '', selectedRepositoryOwner || '', selectedRepositoryName || '', { search: repositoryBranchSearch, limit: 50 }),
    enabled: Boolean(repositoryBindingDialogOpen && selectedRepositoryAccountId && selectedRepositoryOwner && selectedRepositoryName),
  })
  const targetBuildOptions = useQuery({
    queryKey: [
      'git-repository-build-options',
      selectedTargetRepositoryBinding?.gitAccountId,
      selectedTargetRepositoryBinding?.owner,
      selectedTargetRepositoryBinding?.repo,
      selectedTargetRepositoryBinding?.defaultBranch,
    ],
    queryFn: () => api.getGitRepositoryBuildOptions(
      selectedTargetRepositoryBinding?.gitAccountId ?? '',
      selectedTargetRepositoryBinding?.owner ?? '',
      selectedTargetRepositoryBinding?.repo ?? '',
      selectedTargetRepositoryBinding?.defaultBranch,
    ),
    enabled: Boolean(targetDialogOpen && targetSourceType === 'repository' && selectedTargetRepositoryBinding?.gitAccountId && selectedTargetRepositoryBinding.owner && selectedTargetRepositoryBinding.repo),
  })
  const dockerfileSuggestions = useMemo(() => targetBuildOptions.data?.dockerfiles ?? [], [targetBuildOptions.data?.dockerfiles])
  const buildContextSuggestions = useMemo(() => targetBuildOptions.data?.directories ?? [], [targetBuildOptions.data?.directories])
  const buildDirectorySuggestions = buildContextSuggestions.filter(option => option !== '.')
  const dockerfilePathField = targetForm.register('dockerfilePath', { required: true })
  const releaseReadyTargets = useMemo(() => deploymentTargets.filter(target => deploymentTargetCanRelease(target, deployableBuildRuns)), [deployableBuildRuns, deploymentTargets])
  const selectedBuildRun = buildRunMap[form.watch('buildRunId')]
  const latestEditingTargetRelease = editingTarget ? latestReleaseByTarget[deploymentReleaseKey(editingTarget.environmentId, editingTarget.id)] : undefined
  const targetHasRuntimeChanges = editingTarget ? deploymentTargetRuntimeChanged(editingTarget, normalizeDeploymentTargetPayload(watchedTargetValues)) : false
  const targetCanRedeploy = Boolean(editingTarget && latestEditingTargetRelease && normalizeBoolean(watchedTargetValues.enabled, editingTarget.enabled))
  const targetRuntimeFilesValid = targetConfigFilesValid && targetSecretFilesValid
  const copyDeploymentText = (value?: string) => {
    const text = value?.trim()
    if (!text || text === '-')
      return
    navigator.clipboard.writeText(text)
      .then(() => toast.success(t('common.copied')))
      .catch(error => toast.error(error.message))
  }
  const releaseLogs = useQuery({
    queryKey: ['release-logs', projectId, logRelease?.id],
    queryFn: () => api.getReleaseLogs(projectId, logRelease!.id),
    enabled: Boolean(projectId && logRelease),
    refetchInterval: logRelease?.status === 'running' || logRelease?.status === 'pending' ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const runtimeLogs = useQuery({
    queryKey: ['release-runtime-logs', projectId, logRelease?.id],
    queryFn: () => api.getReleaseRuntimeLogs(projectId, logRelease!.id, { tailLines: 500 }),
    enabled: Boolean(projectId && logRelease && logView === 'runtime'),
    refetchInterval: logRelease?.status === 'running' || logRelease?.status === 'pending' ? WORKFLOW_STATUS_REFETCH_INTERVAL_MS : false,
  })
  const runtimeConfigSets = useQuery({
    queryKey: ['runtime-config-sets', projectId],
    queryFn: () => api.listProjectRuntimeConfigSets(projectId),
    enabled: Boolean(projectId),
  })
  const runtimeClusters = useQuery({
    queryKey: ['runtime-clusters', projectId],
    queryFn: () => api.listRuntimeClusters(projectId),
    enabled: Boolean(projectId && deploymentTargets.length > 0),
  })
  const runtimeClusterMap = useMemo(() => Object.fromEntries((runtimeClusters.data ?? []).map(cluster => [cluster.id, cluster])), [runtimeClusters.data])
  const defaultRuntimeCluster = useMemo(() => {
    const clusters = runtimeClusters.data ?? []
    return clusters.find(cluster => cluster.isDefault) ?? clusters[0]
  }, [runtimeClusters.data])
  const workloadClusterIds = useMemo(() => {
    const ids = new Set<string>()
    for (const target of deploymentTargets) {
      const environment = environmentMap[target.environmentId]
      const clusterId = environment?.clusterId?.trim() || defaultRuntimeCluster?.id
      if (clusterId)
        ids.add(clusterId)
    }
    return [...ids].sort()
  }, [defaultRuntimeCluster?.id, deploymentTargets, environmentMap])
  const workloadResourceQueries = useQueries({
    queries: workloadClusterIds.map(clusterId => ({
      enabled: Boolean(projectId && applicationId && clusterId),
      queryFn: () => api.listRuntimeClusterResources(clusterId, { kind: 'workloads', projectId, applicationId }),
      queryKey: ['runtime-cluster-resources', clusterId, 'workloads', projectId, applicationId],
      refetchInterval: WORKFLOW_STATUS_REFETCH_INTERVAL_MS,
    })),
  })
  const serviceResourceQueries = useQueries({
    queries: workloadClusterIds.map(clusterId => ({
      enabled: Boolean(projectId && applicationId && clusterId),
      queryFn: () => api.listRuntimeClusterResources(clusterId, { kind: 'services', projectId, applicationId }),
      queryKey: ['runtime-cluster-resources', clusterId, 'services', projectId, applicationId],
      refetchInterval: WORKFLOW_STATUS_REFETCH_INTERVAL_MS,
    })),
  })
  const workloadResourcesByCluster = useMemo(() => Object.fromEntries(workloadClusterIds.map((clusterId, index) => [clusterId, workloadResourceQueries[index]?.data ?? []] as const)), [workloadClusterIds, workloadResourceQueries])
  const workloadLoadingByCluster = useMemo(() => Object.fromEntries(workloadClusterIds.map((clusterId, index) => [clusterId, Boolean(workloadResourceQueries[index]?.isLoading || workloadResourceQueries[index]?.isFetching)] as const)), [workloadClusterIds, workloadResourceQueries])
  const workloadErrorByCluster = useMemo(() => Object.fromEntries(workloadClusterIds.map((clusterId, index) => [clusterId, Boolean(workloadResourceQueries[index]?.isError)] as const)), [workloadClusterIds, workloadResourceQueries])
  const serviceResourcesByCluster = useMemo(() => Object.fromEntries(workloadClusterIds.map((clusterId, index) => [clusterId, serviceResourceQueries[index]?.data ?? []] as const)), [serviceResourceQueries, workloadClusterIds])
  const deploymentRows = useMemo(() => deploymentTargets.map((target) => {
    const environment = environmentMap[target.environmentId]
    const runtimeCluster = environment?.clusterId ? runtimeClusterMap[environment.clusterId] : defaultRuntimeCluster
    const clusterId = environment?.clusterId?.trim() || runtimeCluster?.id || defaultRuntimeCluster?.id || ''
    return {
      environment,
      internalEndpoint: buildInternalServiceEndpoint(target, serviceResourcesByCluster[clusterId] ?? []),
      release: latestReleaseByTarget[deploymentReleaseKey(target.environmentId, target.id)],
      runtimeStatus: buildDeploymentRuntimeStatus(
        target,
        environment,
        runtimeCluster ?? defaultRuntimeCluster,
        workloadResourcesByCluster,
        workloadLoadingByCluster,
        workloadErrorByCluster,
      ),
      target,
    }
  }), [defaultRuntimeCluster, deploymentTargets, environmentMap, latestReleaseByTarget, runtimeClusterMap, serviceResourcesByCluster, workloadErrorByCluster, workloadLoadingByCluster, workloadResourcesByCluster])
  const runtimeConfigRestartTargets = useMemo(() => {
    if (!runtimeConfigRestartSetId)
      return []
    return deploymentTargets.filter(target => normalizeStringIds(target.runtimeConfigSetIds).includes(runtimeConfigRestartSetId))
  }, [deploymentTargets, runtimeConfigRestartSetId])
  const runtimeConfigRedeployableTargets = useMemo(() => runtimeConfigRestartTargets.filter((target) => {
    const latestRelease = latestReleaseByTarget[deploymentReleaseKey(target.environmentId, target.id)]
    return Boolean(redeployReleasePayload(target, latestRelease))
  }), [latestReleaseByTarget, runtimeConfigRestartTargets])
  const resetTargetForm = (target?: DeploymentTarget | null) => {
    const defaultRegistry = registries.find(registry => registry.credentialSet && registry.isDefault) ?? registries.find(registry => registry.credentialSet) ?? registries.find(registry => registry.isDefault) ?? registries[0]
    const defaultEnvironment = environments[0]
    const defaultBinding = repositoryBindings[0]
    const sourceType = target?.sourceType ?? 'repository'
    targetForm.reset({
      ...deploymentTargetDefaults,
      ...target,
      sourceType,
      environmentId: target?.environmentId ?? defaultEnvironment?.id ?? '',
      repositoryBindingId: target?.repositoryBindingId ?? defaultBinding?.id ?? '',
      targetRegistryId: target?.targetRegistryId ?? defaultRegistry?.id ?? '',
      targetImageRef: deploymentTargetImageRef(target ?? undefined) || defaultTargetImageRef(defaultRegistry, projectSlug, appSlug),
      buildHooksEnabled: target?.buildHooksEnabled ?? true,
      buildHookBindings: target?.buildHookBindings ?? [],
      buildVariableSetIds: normalizeStringIds(target?.buildVariableSetIds),
      runtimeConfigSetIds: normalizeStringIds(target?.runtimeConfigSetIds),
      secretRefs: '',
      secretFiles: '',
      dataRetentionEnabled: target?.dataRetentionEnabled ?? false,
      dataCapacity: target?.dataCapacity || '1Gi',
      dataMountPath: target?.dataMountPath || '/data',
      dataVolumes: target?.dataVolumes || serializeRuntimeDataVolumes(parseRuntimeDataVolumes('', target?.dataMountPath || '/data', target?.dataCapacity || '1Gi')),
      enabled: target?.enabled ?? true,
    })
  }
  const openTargetDialog = (target?: DeploymentTarget) => {
    setEditingTarget(target ?? null)
    setTargetConfigFilesValid(true)
    setTargetSecretFilesValid(true)
    setRuntimeConfigRestartSetId('')
    setRuntimeConfigRestartAffectedCount(0)
    resetTargetForm(target)
    setTargetDialogOpen(true)
  }
  const toggleRuntimeConfigSet = (setId: string, checked: boolean) => {
    const current = new Set(normalizeStringIds(targetForm.getValues('runtimeConfigSetIds')))
    if (checked)
      current.add(setId)
    else
      current.delete(setId)
    targetForm.setValue('runtimeConfigSetIds', Array.from(current), { shouldDirty: true, shouldValidate: true })
  }
  const updateTargetDataVolumes = (rows: typeof targetDataVolumes) => {
    targetForm.setValue('dataVolumes', serializeRuntimeDataVolumes(rows), { shouldDirty: true, shouldValidate: true })
  }
  const openRuntimeConfigDialog = (set?: ProjectRuntimeConfigSet) => {
    setEditingRuntimeConfigSet(set ?? null)
    setRuntimeConfigFilesValid(true)
    setRuntimeSecretFilesValid(true)
    runtimeConfigForm.reset(set
      ? {
          configFiles: set.configFiles,
          enabled: set.enabled,
          envVars: set.envVars,
          name: set.name,
          secretFiles: '',
          secretRefs: '',
        }
      : runtimeConfigDefaults)
    setRuntimeConfigDialogOpen(true)
  }
  const resetRepositoryBindingForm = () => {
    repositoryBindingForm.reset(repositoryBindingDefaults)
    setRepositoryBranchSearch('')
  }
  const openRepositoryBindingDialog = () => {
    resetRepositoryBindingForm()
    setRepositoryBindingDialogOpen(true)
  }
  const openReleaseDialog = (environmentId = '', deploymentTargetId = '') => {
    const defaultTarget = deploymentTargetId
      ? deploymentTargets.find(target => target.id === deploymentTargetId)
      : releaseReadyTargets[0]
    const targetId = defaultTarget?.id ?? deploymentTargetId
    const matchedRun = targetId ? deployableBuildRuns.find(run => run.deploymentTargetId === targetId) : undefined
    form.reset({
      ...releaseDefaults,
      applicationId: matchedRun?.applicationId ?? applicationId,
      deploymentTargetId: targetId ?? '',
      buildRunId: matchedRun?.id ?? '',
      environmentId: defaultTarget?.environmentId ?? environmentId,
      imageRef: matchedRun ? buildRunImageRef(matchedRun) : defaultTarget?.imageRef ?? '',
    })
    setDialogOpen(true)
  }
  useImperativeHandle(ref, () => ({ openReleaseDialog, openTargetDialog: () => openTargetDialog() }))
  useEffect(() => {
    if (!selectedBuildRun)
      return
    form.setValue('deploymentTargetId', selectedBuildRun.deploymentTargetId, { shouldDirty: true, shouldValidate: true })
    form.setValue('applicationId', selectedBuildRun.applicationId, { shouldDirty: true, shouldValidate: true })
    form.setValue('imageRef', buildRunImageRef(selectedBuildRun), { shouldDirty: true, shouldValidate: true })
  }, [form, selectedBuildRun])
  useEffect(() => {
    if (!selectedReleaseTarget || selectedBuildRun)
      return
    form.setValue('environmentId', selectedReleaseTarget.environmentId, { shouldDirty: true, shouldValidate: true })
    form.setValue('applicationId', applicationId, { shouldDirty: true, shouldValidate: true })
    if (selectedReleaseTarget.sourceType === 'image')
      form.setValue('imageRef', selectedReleaseTarget.imageRef, { shouldDirty: true, shouldValidate: true })
  }, [applicationId, form, selectedBuildRun, selectedReleaseTarget])
  useEffect(() => {
    if (!targetDialogOpen || editingTarget || targetSourceType !== 'repository')
      return
    const dockerfilePath = dockerfileSuggestions[0]
    if (!dockerfilePath)
      return
    const currentDockerfile = targetForm.getValues('dockerfilePath')?.trim()
    if (currentDockerfile && currentDockerfile !== 'Dockerfile')
      return
    applyDockerfileBuildDefaults(targetForm, dockerfilePath, buildContextSuggestions)
  }, [buildContextSuggestions, dockerfileSuggestions, editingTarget, targetDialogOpen, targetForm, targetSourceType])
  const createRelease = useMutation({
    mutationFn: (values: ReleaseForm) => api.createRelease(projectId, values),
    onSuccess: () => {
      toast.success(t('deploymentsPage.releaseCreated'))
      setDialogOpen(false)
      form.reset(releaseDefaults)
      queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const rollbackRelease = useMutation({
    mutationFn: (releaseId: string) => api.rollbackRelease(projectId, releaseId),
    onSuccess: () => {
      toast.success(t('deploymentsPage.rollbackQueued'))
      queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteTarget = useMutation({
    mutationFn: (target: DeploymentTarget) => api.deleteDeploymentTarget(projectId, applicationId, target.id),
    onSuccess: () => {
      toast.success(t('deploymentsPage.targetDeleted'))
      setTargetToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['deployment-targets', projectId, applicationId] })
      queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const saveRuntimeConfigSet = useMutation({
    mutationFn: (values: ProjectRuntimeConfigSetPayload) => editingRuntimeConfigSet
      ? api.updateProjectRuntimeConfigSet(projectId, editingRuntimeConfigSet.id, normalizeRuntimeConfigPayload(values))
      : api.createProjectRuntimeConfigSet(projectId, normalizeRuntimeConfigPayload(values)),
    onSuccess: (set) => {
      toast.success(t(editingRuntimeConfigSet ? 'runtimeConfigSets.updated' : 'runtimeConfigSets.created'))
      if (!editingRuntimeConfigSet) {
        toggleRuntimeConfigSet(set.id, true)
      }
      else if ((set.affectedDeploymentTargetCount ?? 0) > 0) {
        setRuntimeConfigRestartSetId(set.id)
        setRuntimeConfigRestartAffectedCount(set.affectedDeploymentTargetCount ?? 0)
      }
      setRuntimeConfigDialogOpen(false)
      setEditingRuntimeConfigSet(null)
      runtimeConfigForm.reset(runtimeConfigDefaults)
      queryClient.invalidateQueries({ queryKey: ['runtime-config-sets', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const redeployRuntimeConfigTargets = useMutation({
    mutationFn: async () => {
      let queued = 0
      let skipped = 0
      for (const target of runtimeConfigRestartTargets) {
        const releasePayload = redeployReleasePayload(target, latestReleaseByTarget[deploymentReleaseKey(target.environmentId, target.id)])
        if (!releasePayload) {
          skipped++
          continue
        }
        await api.createRelease(projectId, releasePayload)
        queued++
      }
      return { queued, skipped }
    },
    onSuccess: ({ queued, skipped }) => {
      toast.success(t('deploymentsPage.runtimeConfigRedeployQueued', { queued, skipped }))
      setRuntimeConfigRestartSetId('')
      setRuntimeConfigRestartAffectedCount(0)
      queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const createRepositoryBinding = useMutation({
    mutationFn: (values: RepositoryBindingForm) => api.createRepositoryBinding(projectId, {
      applicationId,
      autoConfigureWebhook: values.autoConfigureWebhook,
      cloneUrl: values.cloneUrl ?? '',
      defaultBranch: values.defaultBranch || 'main',
      gitAccountId: values.gitAccountId,
      owner: values.owner,
      repo: values.repo,
      webhookStatus: values.webhookStatus,
    }),
    onSuccess: (binding) => {
      toast.success(t('repositories.bindingSaved'))
      queryClient.setQueryData<RepositoryBinding[]>(['repository-bindings', projectId], (items = []) => [
        ...items.filter(item => item.id !== binding.id),
        binding,
      ])
      targetForm.setValue('repositoryBindingId', binding.id, { shouldDirty: true, shouldValidate: true })
      setRepositoryBindingDialogOpen(false)
      resetRepositoryBindingForm()
      queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
      queryClient.invalidateQueries({ queryKey: ['applications', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const saveTarget = useMutation({
    mutationFn: async ({ redeploy, values }: { redeploy: boolean, values: DeploymentTargetPayload }) => {
      const payload = normalizeDeploymentTargetPayload(values)
      const savedTarget = editingTarget
        ? api.updateDeploymentTarget(projectId, applicationId, editingTarget.id, payload)
        : api.createDeploymentTarget(projectId, applicationId, payload)
      const target = await savedTarget
      if (!redeploy)
        return { redeploy, target }
      const releasePayload = redeployReleasePayload(target, latestEditingTargetRelease)
      if (!releasePayload)
        throw new Error(t('deploymentsPage.redeployUnavailable'))
      await api.createRelease(projectId, releasePayload)
      return { redeploy, target }
    },
    onSuccess: ({ redeploy }) => {
      toast.success(t(redeploy ? 'deploymentsPage.targetUpdatedAndRedeployQueued' : editingTarget ? 'deploymentsPage.targetUpdated' : 'deploymentsPage.targetCreated'))
      setTargetDialogOpen(false)
      setEditingTarget(null)
      targetForm.reset(deploymentTargetDefaults)
      queryClient.invalidateQueries({ queryKey: ['deployment-targets', projectId, applicationId] })
      if (redeploy)
        queryClient.invalidateQueries({ queryKey: ['releases', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  return (
    <div className="grid gap-4">
      <DataList
        columns={[
          { key: 'name', header: t('common.name'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => <span className="block max-w-32 truncate" title={item.target.name}>{item.target.name}</span> },
          { key: 'deploymentTarget', header: t('buildsPage.buildConfig'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => <span className="block max-w-32 truncate" title={item.target.name}>{item.target.name}</span> },
          { key: 'environment', header: t('deploymentsPage.environment'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => <span className="block max-w-40 truncate" title={releaseEnvironmentLabel(item.environment, item.target.environmentId, t)}>{releaseEnvironmentLabel(item.environment, item.target.environmentId, t)}</span> },
          { key: 'runtimeData', header: t('deploymentsPage.runtimeData'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => item.target.dataRetentionEnabled ? (item.target.dataCapacity || '1Gi') : t('common.disabled') },
          { key: 'auto', header: t('deploymentsPage.autoDeploy'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => <StatusValueBadge value={item.target.autoDeploy ? 'enabled' : 'disabled'} /> },
          { key: 'runtimeStatus', header: t('deploymentsPage.runtimeStatus'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => <DeploymentRuntimeStatusBadge status={item.runtimeStatus} /> },
          { key: 'internalEndpoint', header: t('deploymentsPage.internalEndpoint'), className: 'min-w-56 px-4 py-3 align-middle', render: item => <InternalServiceEndpoint endpoint={item.internalEndpoint} onCopy={copyDeploymentText} /> },
          { key: 'revision', header: t('deploymentsPage.revision'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => item.release ? `#${item.release.revision}` : '-' },
          { key: 'image', header: t('deploymentsPage.image'), className: 'min-w-48 px-4 py-3 align-middle', render: item => item.release ? <CopyableTruncatedText className="max-w-60 rounded bg-background px-2 py-1 font-mono text-xs" display={shortImageRef(item.release.imageRef)} value={item.release.imageRef} onCopy={copyDeploymentText} /> : '-' },
          { key: 'status', header: t('deploymentsPage.releaseStatus'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => item.release ? <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={item.release.status} /> : <StatusValueBadge label={t('deploymentsPage.notDeployed')} value="pending" /> },
          { key: 'message', header: t('deploymentsPage.rolloutMessage'), className: 'min-w-56 px-4 py-3 align-middle', render: item => <CopyableTruncatedText className="max-w-72 text-sm text-muted-foreground" display={compactReleaseMessage(item.release?.message)} value={item.release?.message} onCopy={copyDeploymentText} /> },
          { key: 'time', header: t('deploymentsPage.releaseTime'), className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle', render: item => item.release ? formatReleaseTime(item.release, t) : '-' },
          {
            key: 'actions',
            header: t('common.actions'),
            cellClassName: 'bg-card',
            className: 'sticky right-0 z-10 w-[1%] whitespace-nowrap px-4 py-3 text-right align-middle shadow-[-10px_0_16px_-16px_rgba(15,23,42,0.6)]',
            headerClassName: 'z-20 bg-muted/95',
            render: item => (
              <div className="flex justify-end">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button aria-label={t('common.actions')} size="icon" variant="ghost">
                      <MoreHorizontal className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem disabled={!deploymentTargetCanRelease(item.target, deployableBuildRuns) || createRelease.isPending} onSelect={() => openReleaseDialog(item.target.environmentId, item.target.id)}>
                      <Package className="size-4" />
                      {item.release ? t('deploymentsPage.createRelease') : t('deploymentsPage.deployToEnvironment')}
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => openTargetDialog(item.target)}>
                      <Pencil className="size-4" />
                      {t('common.edit')}
                    </DropdownMenuItem>
                    {item.release && (
                      <DropdownMenuItem onSelect={() => item.release && setLogRelease(item.release)}>
                        <Eye className="size-4" />
                        {t('deploymentsPage.viewLogs')}
                      </DropdownMenuItem>
                    )}
                    {item.release && (
                      <DropdownMenuItem
                        disabled={item.release.status !== 'succeeded' && item.release.status !== 'running'}
                        onSelect={() => {
                          if (!item.release)
                            return
                          setConsoleRelease(item.release)
                          setConsoleContainer('')
                        }}
                      >
                        <Terminal className="size-4" />
                        {t('deploymentsPage.webConsole')}
                      </DropdownMenuItem>
                    )}
                    {item.release && (
                      <DropdownMenuItem disabled={item.release.status !== 'succeeded' || rollbackRelease.isPending} onSelect={() => item.release && rollbackRelease.mutate(item.release.id)}>
                        <RotateCcw className="size-4" />
                        {t('deploymentsPage.rollback')}
                      </DropdownMenuItem>
                    )}
                    {item.target.dataRetentionEnabled && (
                      <DropdownMenuItem onSelect={() => window.open(deploymentTargetDataExportUrl(projectId, applicationId, item.target.id), '_blank', 'noopener,noreferrer')}>
                        <Download className="size-4" />
                        {t('deploymentsPage.exportData')}
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem disabled={deleteTarget.isPending} variant="destructive" onSelect={() => setTargetToDelete(item.target)}>
                      <Trash2 className="size-4" />
                      {t('common.delete')}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            ),
          },
        ]}
        emptyDescription={t('deploymentsPage.emptyDeploymentsDescription')}
        emptyTitle={t('deploymentsPage.emptyDeployments')}
        items={deploymentRows}
        rowKey={item => item.target.id}
      />
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('deploymentsPage.createRelease')}</DialogTitle>
            <DialogDescription>{t('deploymentsPage.releaseDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={form.handleSubmit(values => createRelease.mutate(values))}>
            {selectedReleaseTarget?.sourceType !== 'image' && (
              <Field hint={t('deploymentsPage.buildRunHint')} label={t('deploymentsPage.buildRun')} required>
                <Select {...form.register('buildRunId', { required: true })}>
                  <option value="">{t('common.select')}</option>
                  {selectableBuildRuns.map(run => <option key={run.id} value={run.id}>{buildRunOptionLabel(run)}</option>)}
                </Select>
              </Field>
            )}
            <Field label={t('buildsPage.buildConfig')}>
              <Select {...form.register('deploymentTargetId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {releaseReadyTargets.map(target => <option key={target.id} value={target.id}>{target.name}</option>)}
              </Select>
            </Field>
            <Field label={t('deploymentsPage.environment')} required>
              <Select {...form.register('environmentId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {environments.map(environment => <option key={environment.id} value={environment.id}>{environment.name}</option>)}
              </Select>
            </Field>
            <Field label={t('deploymentsPage.image')} required><Input {...form.register('imageRef', { required: true })} /></Field>
            <DialogFooter><Button disabled={!form.formState.isValid || createRelease.isPending} type="submit">{t('common.save')}</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog
        open={targetDialogOpen}
        onOpenChange={(open) => {
          setTargetDialogOpen(open)
          if (!open) {
            setEditingTarget(null)
            targetForm.reset(deploymentTargetDefaults)
          }
        }}
      >
        <DialogContent className="flex max-h-[90vh] max-w-4xl flex-col overflow-hidden p-0">
          <DialogHeader className="border-b border-border px-6 py-4">
            <DialogTitle>{editingTarget ? t('deploymentsPage.editDeploymentTarget') : t('deploymentsPage.createDeploymentTarget')}</DialogTitle>
            <DialogDescription>{t('deploymentsPage.deploymentTargetDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="flex min-h-0 flex-1 flex-col" onSubmit={targetForm.handleSubmit(values => saveTarget.mutate({ redeploy: false, values }))}>
            <div className="grid gap-5 overflow-y-auto px-6 py-4 pb-6">
              <div className="grid gap-3 md:grid-cols-2">
                <Field hint={t('deploymentsPage.deploymentConfigNameHint')} label={t('common.name')} required>
                  <Input {...targetForm.register('name', { required: true })} placeholder={t('deploymentsPage.deploymentConfigNamePattern')} />
                </Field>
                <Field label={t('deploymentsPage.environment')} required>
                  <Select {...targetForm.register('environmentId', { required: true })}>
                    <option value="">{t('common.select')}</option>
                    {environments.map(environment => <option key={environment.id} value={environment.id}>{releaseEnvironmentLabel(environment, environment.id, t)}</option>)}
                  </Select>
                </Field>
                <Field hint={t('apps.sourceTypeHint')} label={t('apps.sourceType')} required>
                  <Select {...targetForm.register('sourceType', { required: true })}>
                    <option value="repository">{t('apps.repository')}</option>
                    <option value="image">{t('apps.image')}</option>
                  </Select>
                </Field>
                <Field label={t('common.status')}>
                  <Select {...targetForm.register('enabled')}>
                    <option value="true">{t('common.enabled')}</option>
                    <option value="false">{t('common.disabled')}</option>
                  </Select>
                </Field>
              </div>
              {targetSourceType === 'repository'
                ? (
                    <div className="grid gap-3 md:grid-cols-2">
                      <Field label={t('apps.repository')} required>
                        <div className="flex flex-col gap-2 sm:flex-row">
                          <Select containerClassName="min-w-0 flex-1" {...targetForm.register('repositoryBindingId', { required: targetSourceType === 'repository' })}>
                            <option value="">{t('common.select')}</option>
                            {repositoryBindings.map(binding => (
                              <option key={binding.id} value={binding.id}>
                                {binding.owner}
                                /
                                {binding.repo}
                              </option>
                            ))}
                          </Select>
                          <Button className="shrink-0" type="button" variant="secondary" onClick={openRepositoryBindingDialog}>
                            <Plus className="size-4" />
                            {t('deploymentsPage.bindRepositoryInTarget')}
                          </Button>
                        </div>
                      </Field>
                      <Field label={t('buildsPage.targetRegistry')} required>
                        <Select {...targetForm.register('targetRegistryId', { required: targetSourceType === 'repository' })}>
                          <option value="">{t('common.select')}</option>
                          {registries.map(registry => <option key={registry.id} value={registry.id}>{registryOptionLabel(registry)}</option>)}
                        </Select>
                      </Field>
                      <Field hint={t('buildsPage.dockerfileLookupHint')} label={t('buildsPage.dockerfilePath')} required>
                        <Input
                          {...dockerfilePathField}
                          list="deployment-target-dockerfile-options"
                          placeholder="Dockerfile"
                          onChange={(event) => {
                            dockerfilePathField.onChange(event)
                            applyDockerfileBuildDefaults(targetForm, event.target.value, buildContextSuggestions)
                          }}
                        />
                        <datalist id="deployment-target-dockerfile-options">
                          {dockerfileSuggestions.map(option => <option key={option} value={option} />)}
                        </datalist>
                        {targetBuildOptions.isFetching && <p className="mt-1 text-xs text-muted-foreground">{t('apps.detectingRepository')}</p>}
                        {targetBuildOptions.isError && <p className="mt-1 text-xs text-destructive">{t('deploymentsPage.buildOptionsLoadFailed')}</p>}
                      </Field>
                      <Field hint={t('buildsPage.buildContextLookupHint')} label={t('buildsPage.buildContext')} required>
                        <Input {...targetForm.register('buildContext', { required: true })} list="deployment-target-build-context-options" placeholder="." />
                        <datalist id="deployment-target-build-context-options">
                          {buildContextSuggestions.map(option => <option key={option} value={option} />)}
                        </datalist>
                      </Field>
                      <Field hint={t('buildsPage.buildDirectoryHint')} label={t('buildsPage.buildDirectory')}>
                        <Input {...targetForm.register('buildDirectory')} list="deployment-target-build-directory-options" placeholder={t('buildsPage.buildDirectoryPlaceholder')} />
                        <datalist id="deployment-target-build-directory-options">
                          {buildDirectorySuggestions.map(option => <option key={option} value={option} />)}
                        </datalist>
                      </Field>
                      <Field hint={t('buildsPage.targetImageRefHint')} label={t('buildsPage.targetImageRef')} required>
                        <TargetImageRefInput
                          placeholder={t('buildsPage.targetImageRefPlaceholder')}
                          prefix={targetImagePrefix}
                          register={targetForm.register('targetImageRef', { required: targetSourceType === 'repository' })}
                        />
                      </Field>
                    </div>
                  )
                : (
                    <Field hint={t('apps.imageReferenceHint')} label={t('apps.imageReference')} required>
                      <Input {...targetForm.register('imageRef', { required: targetSourceType === 'image' })} placeholder={t('apps.imageReferencePlaceholder')} />
                    </Field>
                  )}
              <div className="grid gap-3 md:grid-cols-2">
                <Field hint={t('deploymentsPage.branchPatternHint')} label={t('deploymentsPage.branchPattern')}>
                  <Input {...targetForm.register('branchPattern')} placeholder="main,release-*" />
                </Field>
                <Field hint={t('deploymentsPage.tagPatternHint')} label={t('deploymentsPage.tagPattern')}>
                  <Input {...targetForm.register('tagPattern')} placeholder="v*" />
                </Field>
                <Field hint={t('apps.buildConcurrencyPolicyHint')} label={t('apps.buildConcurrencyPolicy')}>
                  <Select {...targetForm.register('concurrencyPolicy')}>
                    <option value="queue">{t('apps.buildConcurrencyPolicies.queue')}</option>
                    <option value="parallel">{t('apps.buildConcurrencyPolicies.parallel')}</option>
                  </Select>
                </Field>
                <Field label={t('deploymentsPage.autoDeploy')}>
                  <Select {...targetForm.register('autoDeploy')}>
                    <option value="false">{t('common.disabled')}</option>
                    <option value="true">{t('common.enabled')}</option>
                  </Select>
                </Field>
              </div>
              <div className="grid gap-3">
                <div>
                  <h3 className="text-sm font-semibold">{t('deploymentsPage.runtimeData')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{t('deploymentsPage.runtimeDataDescription')}</p>
                </div>
                <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
                  <Field hint={t('deploymentsPage.dataRetentionHint')} label={t('deploymentsPage.dataRetention')}>
                    <Select {...targetForm.register('dataRetentionEnabled')}>
                      <option value="false">{t('common.disabled')}</option>
                      <option value="true">{t('common.enabled')}</option>
                    </Select>
                  </Field>
                  <Field hint={t('deploymentsPage.dataVolumesHint')} label={t('deploymentsPage.dataVolumes')} required={targetDataRetentionEnabled}>
                    <div className="grid gap-2 rounded-md border border-input bg-background p-3">
                      <div className="hidden gap-2 px-1 text-xs font-medium text-muted-foreground md:grid md:grid-cols-[minmax(7rem,0.7fr)_minmax(0,1.5fr)_minmax(7rem,0.7fr)_auto]">
                        <span>{t('deploymentsPage.dataVolumeName')}</span>
                        <span>{t('deploymentsPage.dataMountPath')}</span>
                        <span>{t('deploymentsPage.dataCapacity')}</span>
                        <span className="sr-only">{t('common.actions')}</span>
                      </div>
                      {targetDataVolumes.map((volume, index) => (
                        <div key={volume.id} className="grid gap-2 md:grid-cols-[minmax(7rem,0.7fr)_minmax(0,1.5fr)_minmax(7rem,0.7fr)_auto]">
                          <Input
                            disabled={!targetDataRetentionEnabled}
                            placeholder={t('deploymentsPage.dataVolumeNamePlaceholder')}
                            value={volume.name}
                            onChange={(event) => {
                              const rows = [...targetDataVolumes]
                              rows[index] = { ...volume, name: event.target.value }
                              updateTargetDataVolumes(rows)
                            }}
                          />
                          <Input
                            disabled={!targetDataRetentionEnabled}
                            placeholder={t('deploymentsPage.dataMountPathPlaceholder')}
                            value={volume.mountPath}
                            onChange={(event) => {
                              const rows = [...targetDataVolumes]
                              rows[index] = { ...volume, mountPath: event.target.value }
                              updateTargetDataVolumes(rows)
                            }}
                          />
                          <Input
                            disabled={!targetDataRetentionEnabled}
                            placeholder={t('deploymentsPage.dataCapacityPlaceholder')}
                            value={volume.capacity}
                            onChange={(event) => {
                              const rows = [...targetDataVolumes]
                              rows[index] = { ...volume, capacity: event.target.value }
                              updateTargetDataVolumes(rows)
                            }}
                          />
                          <Button
                            aria-label={t('deploymentsPage.removeDataVolume')}
                            disabled={!targetDataRetentionEnabled || targetDataVolumes.length <= 1}
                            size="icon"
                            type="button"
                            variant="ghost"
                            onClick={() => updateTargetDataVolumes(targetDataVolumes.filter(row => row.id !== volume.id))}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      ))}
                      <div>
                        <Button
                          disabled={!targetDataRetentionEnabled}
                          size="sm"
                          type="button"
                          variant="secondary"
                          onClick={() => updateTargetDataVolumes([...targetDataVolumes, emptyRuntimeDataVolumeRow(targetDataVolumes.length)])}
                        >
                          <Plus className="size-4" />
                          {t('deploymentsPage.addDataVolume')}
                        </Button>
                      </div>
                    </div>
                  </Field>
                </div>
              </div>
              <div className="grid gap-3">
                <div>
                  <h3 className="text-sm font-semibold">{t('deploymentsPage.runtimeConfig')}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{t('deploymentsPage.runtimeConfigDescription')}</p>
                </div>
                <Field hint={t('deploymentsPage.runtimeConfigSetsHint')} label={t('deploymentsPage.runtimeConfigSets')}>
                  <div className="grid gap-3 rounded-md border border-input bg-background p-3">
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-sm font-medium text-foreground">{t('deploymentsPage.runtimeConfigSets')}</span>
                      <Button size="sm" type="button" variant="secondary" onClick={() => openRuntimeConfigDialog()}>
                        <FileCode2 className="size-4" />
                        {t('runtimeConfigSets.createTitle')}
                      </Button>
                    </div>
                    {(runtimeConfigSets.data ?? []).length > 0
                      ? (runtimeConfigSets.data ?? []).map(set => (
                          <div key={set.id} className="flex items-center justify-between gap-3 rounded-md px-2 py-1.5 text-sm hover:bg-muted/60">
                            <label className="flex min-w-0 flex-1 items-center gap-3">
                              <input
                                checked={selectedRuntimeConfigSetIds.includes(set.id)}
                                className="size-4 shrink-0 accent-primary"
                                disabled={!set.enabled}
                                type="checkbox"
                                onChange={event => toggleRuntimeConfigSet(set.id, event.target.checked)}
                              />
                              <span className="min-w-0">
                                <span className="block truncate font-medium" title={set.name}>{set.name}</span>
                                <span className="block truncate text-xs text-muted-foreground">{set.enabled ? t('common.enabled') : t('common.disabled')}</span>
                              </span>
                            </label>
                            <Button aria-label={t('runtimeConfigSets.editTitle')} size="sm" type="button" variant="ghost" onClick={() => openRuntimeConfigDialog(set)}>
                              <Pencil className="size-4" />
                            </Button>
                          </div>
                        ))
                      : <p className="text-sm text-muted-foreground">{t('deploymentsPage.emptyRuntimeConfigSets')}</p>}
                  </div>
                </Field>
                {runtimeConfigRestartAffectedCount > 0 && (
                  <div className="flex gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-amber-950 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-100">
                    <Rocket className="mt-0.5 size-4 shrink-0" />
                    <div className="grid flex-1 gap-2 text-sm">
                      <div className="grid gap-1">
                        <p className="font-medium">{t('deploymentsPage.runtimeConfigSetChangedTitle')}</p>
                        <p className="text-amber-900/80 dark:text-amber-100/80">
                          {t('deploymentsPage.runtimeConfigSetChangedDescription', {
                            count: runtimeConfigRestartAffectedCount,
                            redeployable: runtimeConfigRedeployableTargets.length,
                          })}
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          disabled={runtimeConfigRedeployableTargets.length === 0 || redeployRuntimeConfigTargets.isPending}
                          size="sm"
                          type="button"
                          variant="secondary"
                          onClick={() => redeployRuntimeConfigTargets.mutate()}
                        >
                          <Rocket className="size-4" />
                          {t('deploymentsPage.redeployAffectedRuntimeConfig')}
                        </Button>
                        <Button
                          size="sm"
                          type="button"
                          variant="ghost"
                          onClick={() => {
                            setRuntimeConfigRestartSetId('')
                            setRuntimeConfigRestartAffectedCount(0)
                          }}
                        >
                          {t('common.close')}
                        </Button>
                      </div>
                    </div>
                  </div>
                )}
                <Field hint={t('deploymentsPage.runtimeEnvVarsHint')} label={t('deploymentsPage.runtimeEnvVars')}>
                  <textarea className="min-h-24 rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20" {...targetForm.register('envVars')} placeholder={t('deploymentsPage.runtimeEnvVarsPlaceholder')} />
                </Field>
                <Field hint={t('deploymentsPage.runtimeConfigRefsHint')} label={t('deploymentsPage.runtimeConfigRefs')}>
                  <textarea className="min-h-24 rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20" {...targetForm.register('configRefs')} placeholder={t('deploymentsPage.runtimeConfigRefsPlaceholder')} />
                </Field>
                <Field hint={t('deploymentsPage.runtimeConfigFilesHint')} label={t('deploymentsPage.runtimeConfigFiles')}>
                  <RuntimeConfigFilesEditor
                    key={`${editingTarget?.id ?? 'new'}-config-files`}
                    initialValue={targetForm.getValues('configFiles') ?? ''}
                    onChange={value => targetForm.setValue('configFiles', value, { shouldDirty: true, shouldValidate: true })}
                    onValidationChange={setTargetConfigFilesValid}
                  />
                </Field>
                <Field hint={editingTarget?.secretRefsSet ? t('deploymentsPage.runtimeSecretRefsConfigured') : t('deploymentsPage.runtimeSecretRefsHint')} label={t('deploymentsPage.runtimeSecretRefs')}>
                  <textarea className="min-h-24 rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition focus-visible:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary/20" {...targetForm.register('secretRefs')} placeholder={t('deploymentsPage.runtimeSecretRefsPlaceholder')} />
                </Field>
                <Field hint={editingTarget?.secretFilesSet ? t('deploymentsPage.runtimeSecretFilesConfigured') : t('deploymentsPage.runtimeSecretFilesHint')} label={t('deploymentsPage.runtimeSecretFiles')}>
                  <RuntimeConfigFilesEditor
                    key={`${editingTarget?.id ?? 'new'}-secret-files`}
                    initialValue={targetForm.getValues('secretFiles') ?? ''}
                    onChange={value => targetForm.setValue('secretFiles', value, { shouldDirty: true, shouldValidate: true })}
                    onValidationChange={setTargetSecretFilesValid}
                  />
                </Field>
              </div>
              {targetHasRuntimeChanges && (
                <div className="flex gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-amber-950 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-100">
                  <Rocket className="mt-0.5 size-4 shrink-0" />
                  <div className="grid gap-1 text-sm">
                    <p className="font-medium">{t('deploymentsPage.runtimeChangesNeedRedeployTitle')}</p>
                    <p className="text-amber-900/80 dark:text-amber-100/80">
                      {targetCanRedeploy ? t('deploymentsPage.runtimeChangesNeedRedeployDescription') : t('deploymentsPage.runtimeChangesNeedRedeployUnavailable')}
                    </p>
                  </div>
                </div>
              )}
            </div>
            <DialogFooter className="shrink-0 border-t border-border bg-background px-6 py-4">
              {targetHasRuntimeChanges && (
                <Button
                  disabled={!targetRuntimeFilesValid || !targetCanRedeploy || saveTarget.isPending}
                  type="button"
                  variant="secondary"
                  onClick={targetForm.handleSubmit(values => saveTarget.mutate({ redeploy: true, values }))}
                >
                  <Rocket className="size-4" />
                  {t('deploymentsPage.saveAndRedeploy')}
                </Button>
              )}
              <Button disabled={!targetRuntimeFilesValid || saveTarget.isPending} type="submit">
                <Save className="size-4" />
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog
        open={repositoryBindingDialogOpen}
        onOpenChange={(open) => {
          setRepositoryBindingDialogOpen(open)
          if (!open)
            resetRepositoryBindingForm()
        }}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t('repositories.bindRepoTitle')}</DialogTitle>
            <DialogDescription>{t('deploymentsPage.repositoryBindingDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={repositoryBindingForm.handleSubmit(values => createRepositoryBinding.mutate(values))}>
            <GitRepositoryPicker
              accounts={gitAccounts.data ?? []}
              providers={gitProviders.data ?? []}
              value={{
                gitAccountId: repositoryBindingForm.watch('gitAccountId') || '',
                owner: repositoryBindingForm.watch('owner') || '',
                repo: repositoryBindingForm.watch('repo') || '',
                cloneUrl: repositoryBindingForm.watch('cloneUrl') || '',
                defaultBranch: repositoryBindingForm.watch('defaultBranch') || 'main',
              }}
              onChange={(next) => {
                repositoryBindingForm.setValue('gitAccountId', next.gitAccountId, { shouldDirty: true, shouldValidate: true })
                repositoryBindingForm.setValue('owner', next.owner, { shouldDirty: true, shouldValidate: true })
                repositoryBindingForm.setValue('repo', next.repo, { shouldDirty: true, shouldValidate: true })
                repositoryBindingForm.setValue('cloneUrl', next.cloneUrl, { shouldDirty: true, shouldValidate: true })
                repositoryBindingForm.setValue('defaultBranch', next.defaultBranch || 'main', { shouldDirty: true, shouldValidate: true })
                setRepositoryBranchSearch('')
              }}
            />
            <div className="grid gap-3 md:grid-cols-3">
              <Field error={repositoryBindingForm.formState.errors.owner?.message} label={t('repositories.owner')} required>
                <Input {...repositoryBindingForm.register('owner')} aria-invalid={Boolean(repositoryBindingForm.formState.errors.owner)} placeholder={t('repositories.ownerPlaceholder')} />
              </Field>
              <Field error={repositoryBindingForm.formState.errors.repo?.message} label={t('repositories.repo')} required>
                <Input {...repositoryBindingForm.register('repo')} aria-invalid={Boolean(repositoryBindingForm.formState.errors.repo)} placeholder={t('repositories.repoPlaceholder')} />
              </Field>
              <Field error={repositoryBindingForm.formState.errors.defaultBranch?.message} label={t('repositories.defaultBranch')}>
                <SearchSelect
                  disabled={!selectedRepositoryAccountId || !selectedRepositoryOwner || !selectedRepositoryName}
                  emptyLabel={t('repositories.noBranches')}
                  limited={repositoryBranches.data?.limited}
                  loading={repositoryBranches.isFetching}
                  options={branchOptions(repositoryBranches.data?.items ?? [], repositoryBindingForm.watch('defaultBranch'))}
                  placeholder={t('repositories.defaultBranchPlaceholder')}
                  search={repositoryBranchSearch}
                  value={repositoryBindingForm.watch('defaultBranch') || ''}
                  onSearchChange={setRepositoryBranchSearch}
                  onValueChange={value => repositoryBindingForm.setValue('defaultBranch', value, { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <Field error={repositoryBindingForm.formState.errors.cloneUrl?.message} label={t('repositories.cloneUrl')}>
                <Input {...repositoryBindingForm.register('cloneUrl')} aria-invalid={Boolean(repositoryBindingForm.formState.errors.cloneUrl)} placeholder={t('repositories.cloneUrlPlaceholder')} />
              </Field>
              <label className="flex items-start gap-3 rounded-md border border-border bg-muted/30 p-3">
                <input className="mt-1 size-4 accent-primary" type="checkbox" {...repositoryBindingForm.register('autoConfigureWebhook')} />
                <span className="grid gap-1 text-sm">
                  <span className="font-medium text-foreground">{t('repositories.autoConfigureWebhook')}</span>
                  <span className="text-xs leading-5 text-muted-foreground">{t('repositories.autoConfigureWebhookHint')}</span>
                </span>
              </label>
            </div>
            <DialogFooter>
              <Button disabled={createRepositoryBinding.isPending || (gitAccounts.data ?? []).length === 0 || !repositoryBindingForm.formState.isValid} type="submit">
                <Plus className="size-4" />
                {t('repositories.saveBinding')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog
        open={runtimeConfigDialogOpen}
        onOpenChange={(open) => {
          setRuntimeConfigDialogOpen(open)
          if (!open) {
            setEditingRuntimeConfigSet(null)
            runtimeConfigForm.reset(runtimeConfigDefaults)
          }
        }}
      >
        <DialogContent className="max-h-[88vh] max-w-3xl overflow-hidden p-0">
          <DialogHeader className="border-b border-border px-6 py-5">
            <DialogTitle>{editingRuntimeConfigSet ? t('runtimeConfigSets.editTitle') : t('runtimeConfigSets.createTitle')}</DialogTitle>
            <DialogDescription>{t('runtimeConfigSets.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid max-h-[calc(88vh-96px)] grid-rows-[minmax(0,1fr)_auto]" onSubmit={runtimeConfigForm.handleSubmit(values => saveRuntimeConfigSet.mutate(values))}>
            <div className="grid gap-4 overflow-y-auto px-6 py-5">
              <Field label={t('common.name')} required><Input {...runtimeConfigForm.register('name', { required: true })} /></Field>
              <Field hint={t('runtimeConfigSets.envVarsHint')} label={t('runtimeConfigSets.envVars')}>
                <Textarea className="min-h-24 font-mono text-sm" {...runtimeConfigForm.register('envVars')} placeholder={t('runtimeConfigSets.envVarsPlaceholder')} />
              </Field>
              <Field hint={t('runtimeConfigSets.configFilesHint')} label={t('runtimeConfigSets.configFiles')}>
                <RuntimeConfigFilesEditor
                  key={`${editingRuntimeConfigSet?.id ?? 'new'}-target-config-files`}
                  initialValue={runtimeConfigForm.getValues('configFiles') ?? ''}
                  onChange={value => runtimeConfigForm.setValue('configFiles', value, { shouldDirty: true, shouldValidate: true })}
                  onValidationChange={setRuntimeConfigFilesValid}
                />
              </Field>
              <Field hint={editingRuntimeConfigSet?.secretRefsSet ? t('runtimeConfigSets.secretRefsConfiguredHint') : t('runtimeConfigSets.secretRefsHint')} label={t('runtimeConfigSets.secretRefs')}>
                <Textarea className="min-h-24 font-mono text-sm" {...runtimeConfigForm.register('secretRefs')} placeholder={t('runtimeConfigSets.secretRefsPlaceholder')} />
              </Field>
              <Field hint={editingRuntimeConfigSet?.secretFilesSet ? t('runtimeConfigSets.secretFilesConfiguredHint') : t('runtimeConfigSets.secretFilesHint')} label={t('runtimeConfigSets.secretFiles')}>
                <RuntimeConfigFilesEditor
                  key={`${editingRuntimeConfigSet?.id ?? 'new'}-target-secret-files`}
                  initialValue={runtimeConfigForm.getValues('secretFiles') ?? ''}
                  onChange={value => runtimeConfigForm.setValue('secretFiles', value, { shouldDirty: true, shouldValidate: true })}
                  onValidationChange={setRuntimeSecretFilesValid}
                />
              </Field>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input className="size-4 accent-primary" type="checkbox" {...runtimeConfigForm.register('enabled')} />
                {t('common.enabled')}
              </label>
            </div>
            <DialogFooter className="border-t border-border bg-background px-6 py-4">
              <Button disabled={!runtimeConfigFilesValid || !runtimeSecretFilesValid || saveRuntimeConfigSet.isPending} type="submit">
                <FileCode2 className="size-4" />
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog open={Boolean(logRelease)} onOpenChange={open => !open && setLogRelease(null)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>{t('deploymentsPage.releaseLogs')}</DialogTitle>
            <DialogDescription>{logRelease?.id}</DialogDescription>
          </DialogHeader>
          <Tabs className="gap-3" value={logView} onValueChange={value => setLogView(value as 'deployment' | 'runtime')}>
            <SegmentedTabsList
              items={(['deployment', 'runtime'] as const).map(view => ({
                label: t(`deploymentsPage.logViews.${view}`),
                value: view,
              }))}
              layoutId="release-log-view-active"
              value={logView}
            />
            <TabsContent value="deployment">
              <pre className="max-h-[60vh] overflow-auto rounded-md border border-border bg-muted p-3 text-xs leading-relaxed text-foreground">
                {releaseLogs.data?.content || t('deploymentsPage.emptyLogs')}
              </pre>
            </TabsContent>
            <TabsContent className="grid gap-3" value="runtime">
              {runtimeLogs.data && (
                <div className="text-xs text-muted-foreground">
                  {t('deploymentsPage.runtimeLogSource', { pod: runtimeLogs.data.pod, container: runtimeLogs.data.container })}
                </div>
              )}
              <pre className="max-h-[60vh] overflow-auto rounded-md border border-border bg-muted p-3 text-xs leading-relaxed text-foreground">
                {runtimeLogs.data?.content || (runtimeLogs.isLoading ? t('common.loading') : t('deploymentsPage.emptyLogs'))}
              </pre>
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(consoleRelease)}
        onOpenChange={(open) => {
          if (!open) {
            setConsoleRelease(null)
          }
        }}
      >
        <DialogContent className="max-w-5xl p-0">
          <DialogHeader>
            <div className="border-b border-border px-5 py-4">
              <DialogTitle>{t('deploymentsPage.webConsole')}</DialogTitle>
              <DialogDescription>{t('deploymentsPage.webConsoleDescription')}</DialogDescription>
            </div>
          </DialogHeader>
          <div className="grid gap-4 px-5 pb-5">
            <div className="overflow-hidden rounded-md border border-zinc-800 bg-zinc-950 text-zinc-100 shadow-xl">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-800 bg-zinc-900 px-4 py-2">
                <div className="flex items-center gap-2">
                  <span className="size-3 rounded-full bg-red-500" />
                  <span className="size-3 rounded-full bg-yellow-400" />
                  <span className="size-3 rounded-full bg-emerald-500" />
                  <span className="ml-2 font-mono text-xs text-zinc-400">{consoleRelease?.id ?? '-'}</span>
                </div>
                <label className="flex min-w-0 items-center gap-2 font-mono text-xs text-zinc-400">
                  <span>{t('deploymentsPage.container')}</span>
                  <input
                    className="h-7 w-32 rounded border border-zinc-700 bg-zinc-950 px-2 text-zinc-100 outline-none transition placeholder:text-zinc-600 focus:border-emerald-500"
                    placeholder={t('deploymentsPage.webConsoleContainerPlaceholder')}
                    value={consoleContainer}
                    onChange={event => setConsoleContainer(event.target.value)}
                  />
                </label>
              </div>
              <RuntimeTerminalPanel container={consoleContainer} projectId={projectId} release={consoleRelease} />
            </div>
          </div>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('deploymentsPage.deleteDeploymentConfigDescription')}
        open={Boolean(targetToDelete)}
        pending={deleteTarget.isPending}
        title={t('deploymentsPage.deleteDeploymentConfigTitle')}
        onConfirm={() => targetToDelete && deleteTarget.mutate(targetToDelete)}
        onOpenChange={open => !open && setTargetToDelete(null)}
      />
    </div>
  )
}

interface DeploymentRuntimeStatus {
  clusterName?: string
  podCount: number
  summary: string
  value: string
}

interface InternalServiceEndpointValue {
  fqdn: string
  namespace: string
  serviceName: string
}

function InternalServiceEndpoint({ endpoint, onCopy }: { endpoint?: InternalServiceEndpointValue, onCopy: (value?: string) => void }) {
  const { t } = useTranslation()
  if (!endpoint)
    return <span className="text-sm text-muted-foreground">-</span>

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button className="grid min-w-0 max-w-64 gap-0.5 text-left transition hover:text-primary" type="button" onClick={() => onCopy(endpoint.fqdn)}>
          <span className="truncate font-mono text-xs">{endpoint.serviceName}</span>
          <span className="truncate text-xs text-muted-foreground">{endpoint.fqdn}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent className="grid max-w-96 gap-1 break-all leading-5" side="top">
        <span>{t('deploymentsPage.internalEndpointHint')}</span>
        <span className="font-mono">{endpoint.fqdn}</span>
      </TooltipContent>
    </Tooltip>
  )
}

function DeploymentRuntimeStatusBadge({ status }: { status: DeploymentRuntimeStatus }) {
  const { t } = useTranslation()
  const detail = status.summary.trim() || t(`deploymentsPage.runtimeStatusDetails.${status.value}`, { defaultValue: '' })
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">
          <StatusValueBadge labelKeyPrefix="deploymentsPage.runtimeStatuses" value={status.value} />
        </span>
      </TooltipTrigger>
      <TooltipContent className="grid max-w-96 gap-1 leading-5" side="top">
        {status.clusterName && <span>{t('deploymentsPage.runtimeStatusCluster', { cluster: status.clusterName })}</span>}
        {status.podCount > 0 && <span>{t('deploymentsPage.runtimePodCount', { count: status.podCount })}</span>}
        {detail && <span className="break-words">{detail}</span>}
      </TooltipContent>
    </Tooltip>
  )
}

function buildDeploymentRuntimeStatus(
  target: DeploymentTarget,
  environment: Environment | undefined,
  runtimeCluster: RuntimeCluster | undefined,
  resourcesByCluster: Record<string, ClusterResource[]>,
  loadingByCluster: Record<string, boolean>,
  errorByCluster: Record<string, boolean>,
): DeploymentRuntimeStatus {
  const clusterId = environment?.clusterId?.trim() || runtimeCluster?.id
  const clusterName = runtimeCluster?.name
  if (!clusterId)
    return { clusterName, podCount: 0, summary: '', value: 'not-configured' }
  if (errorByCluster[clusterId])
    return { clusterName, podCount: 0, summary: '', value: 'unavailable' }
  if (loadingByCluster[clusterId])
    return { clusterName, podCount: 0, summary: '', value: 'checking' }

  const resources = (resourcesByCluster[clusterId] ?? []).filter(resource => resource.deploymentTargetId === target.id)
  const pods = resources.filter(resource => resource.kind.toLowerCase() === 'pod')
  const deployments = resources.filter(resource => resource.kind.toLowerCase() === 'deployment')
  if (resources.length === 0)
    return { clusterName, podCount: 0, summary: '', value: 'not-found' }

  const podStatus = aggregatePodRuntimeStatus(pods)
  if (podStatus)
    return { clusterName, ...podStatus }

  const deployment = deployments[0]
  if (!deployment)
    return { clusterName, podCount: 0, summary: '', value: 'unknown' }
  return {
    clusterName,
    podCount: 0,
    summary: deployment.summary,
    value: normalizeDeploymentRuntimeStatus(deployment.status),
  }
}

function buildInternalServiceEndpoint(target: DeploymentTarget, resources: ClusterResource[]): InternalServiceEndpointValue | undefined {
  const service = resources.find(resource => resource.kind.toLowerCase() === 'service' && resource.deploymentTargetId === target.id)
  const serviceName = service?.name.trim()
  const namespace = service?.namespace.trim()
  if (!serviceName || !namespace)
    return undefined

  return {
    fqdn: `${serviceName}.${namespace}.svc.cluster.local`,
    namespace,
    serviceName,
  }
}

function aggregatePodRuntimeStatus(pods: ClusterResource[]): Omit<DeploymentRuntimeStatus, 'clusterName'> | null {
  if (pods.length === 0)
    return null

  const details = pods.map(pod => ({
    pod,
    value: normalizePodRuntimeStatus(pod),
  }))
  const priority = [
    'crash-loop-back-off',
    'image-pull-back-off',
    'err-image-pull',
    'create-container-config-error',
    'create-container-error',
    'failed',
    'pending',
    'container-creating',
    'not-ready',
    'running',
    'ready',
    'succeeded',
    'unknown',
  ]
  const selected = [...details].sort((left, right) => priority.indexOf(left.value) - priority.indexOf(right.value))[0]
  return {
    podCount: pods.length,
    summary: selected?.pod.summary || '',
    value: selected?.value || 'unknown',
  }
}

function normalizePodRuntimeStatus(pod: ClusterResource) {
  const status = pod.status.trim().toLowerCase()
  const summary = pod.summary.trim().toLowerCase()
  if (summary.includes('crashloopbackoff'))
    return 'crash-loop-back-off'
  if (summary.includes('imagepullbackoff'))
    return 'image-pull-back-off'
  if (summary.includes('errimagepull'))
    return 'err-image-pull'
  if (summary.includes('createcontainerconfigerror'))
    return 'create-container-config-error'
  if (summary.includes('createcontainererror'))
    return 'create-container-error'
  if (summary.includes('containercreating'))
    return 'container-creating'
  if (status === 'failed')
    return 'failed'
  if (status === 'succeeded')
    return 'succeeded'
  if (status === 'pending')
    return 'pending'
  if (status === 'running' && summary.includes('ready 1/1'))
    return 'ready'
  if (status === 'running')
    return 'not-ready'
  return status || 'unknown'
}

function normalizeDeploymentRuntimeStatus(status: string) {
  const value = status.trim().toLowerCase()
  if (value === 'ready')
    return 'ready'
  if (value === 'progressing')
    return 'progressing'
  if (value === 'failed')
    return 'failed'
  return value || 'unknown'
}

function releaseEnvironmentLabel(environment: { name: string, stage?: string } | undefined, environmentID: string, t: ReturnType<typeof useTranslation>['t']) {
  if (!environment)
    return environmentID || '-'
  const stage = environment.stage ? t(`deploymentsPage.stageLabels.${environment.stage}`, { defaultValue: environment.stage }) : ''
  return stage ? `${environment.name} · ${stage}` : environment.name
}

function RuntimeTerminalPanel({ container, projectId, release }: { container: string, projectId: string, release: Release | null }) {
  const { t } = useTranslation()
  const terminalRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!release || !projectId || !terminalRef.current)
      return

    const terminal = new XTerm({
      cursorBlink: true,
      convertEol: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      scrollback: 3000,
      theme: {
        background: '#020617',
        black: '#020617',
        blue: '#60a5fa',
        brightBlack: '#475569',
        brightBlue: '#93c5fd',
        brightCyan: '#67e8f9',
        brightGreen: '#86efac',
        brightMagenta: '#f0abfc',
        brightRed: '#fca5a5',
        brightWhite: '#f8fafc',
        brightYellow: '#fde68a',
        cursor: '#34d399',
        cyan: '#22d3ee',
        foreground: '#e2e8f0',
        green: '#22c55e',
        magenta: '#d946ef',
        red: '#ef4444',
        white: '#cbd5e1',
        yellow: '#facc15',
      },
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(terminalRef.current)
    terminal.writeln(t('deploymentsPage.webConsoleConnecting'))
    terminal.focus()

    const socket = new WebSocket(releaseRuntimeTerminalUrl(projectId, release.id, container))
    socket.binaryType = 'arraybuffer'

    const sendResize = () => {
      if (socket.readyState !== WebSocket.OPEN)
        return
      socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
    }

    const fitAndResize = () => {
      fitAddon.fit()
      sendResize()
    }

    const dataSubscription = terminal.onData((data) => {
      if (socket.readyState === WebSocket.OPEN)
        socket.send(data)
    })
    const resizeObserver = new ResizeObserver(fitAndResize)
    resizeObserver.observe(terminalRef.current)

    const handleOpen = () => {
      fitAndResize()
      terminal.writeln(t('deploymentsPage.webConsoleConnected'))
    }
    const handleMessage = (event: MessageEvent) => {
      if (typeof event.data === 'string') {
        terminal.write(event.data)
        return
      }
      terminal.write(new Uint8Array(event.data))
    }
    const handleClose = () => {
      terminal.writeln('')
      terminal.writeln(t('deploymentsPage.webConsoleDisconnected'))
    }
    const handleError = () => {
      terminal.writeln('')
      terminal.writeln(t('deploymentsPage.webConsoleConnectionFailed'))
    }

    socket.addEventListener('open', handleOpen)
    socket.addEventListener('message', handleMessage)
    socket.addEventListener('close', handleClose)
    socket.addEventListener('error', handleError)

    const fitTimer = window.setTimeout(fitAndResize, 50)
    window.addEventListener('resize', fitAndResize)

    return () => {
      window.clearTimeout(fitTimer)
      window.removeEventListener('resize', fitAndResize)
      socket.removeEventListener('open', handleOpen)
      socket.removeEventListener('message', handleMessage)
      socket.removeEventListener('close', handleClose)
      socket.removeEventListener('error', handleError)
      resizeObserver.disconnect()
      dataSubscription.dispose()
      socket.close()
      terminal.dispose()
    }
  }, [container, projectId, release, t])

  return (
    <div className="p-3">
      <div ref={terminalRef} className="h-[28rem] overflow-hidden rounded border border-zinc-800 bg-slate-950 p-2" />
    </div>
  )
}

function gatewayDeploymentTargetLabel(target: DeploymentTarget, environments: Array<{ id: string, name: string, stage?: string }>, t: ReturnType<typeof useTranslation>['t']) {
  const environment = environments.find(item => item.id === target.environmentId)
  return `${target.name} · ${releaseEnvironmentLabel(environment, target.environmentId, t)}`
}

function deploymentReleaseKey(environmentId: string, deploymentTargetId: string) {
  return `${environmentId}:${deploymentTargetId}`
}

function shortImageRef(imageRef: string) {
  const value = imageRef.trim()
  if (!value)
    return '-'
  const [repository, tag = ''] = value.split(':')
  const parts = repository.split('/').filter(Boolean)
  const compactRepository = parts.length > 2 ? `${parts.at(-2)}/${parts.at(-1)}` : repository
  return tag ? `${compactRepository}:${tag}` : compactRepository
}

function compactReleaseMessage(message?: string) {
  const value = message?.trim()
  if (!value)
    return '-'
  if (value.startsWith('invalid configuration'))
    return 'config invalid'
  if (value.includes('timed out'))
    return 'rollout timeout'
  if (value.includes('Deployment/Service/ConfigMap/Secret'))
    return 'resources applied'
  return value
}

function CopyableTruncatedText({ className, display, value, onCopy }: {
  className?: string
  display: string
  value?: string
  onCopy: (value?: string) => void
}) {
  const title = value?.trim() || display
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          className={`block min-w-0 truncate text-left transition hover:text-primary ${className ?? ''}`}
          type="button"
          onClick={() => onCopy(value)}
        >
          {display}
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-96 break-all leading-5" side="top">
        {title}
      </TooltipContent>
    </Tooltip>
  )
}

function formatReleaseTime(release: Release, t: ReturnType<typeof useTranslation>['t']) {
  if (release.finishedAt)
    return formatSmartDateTime(release.finishedAt, t)
  if (release.startedAt)
    return formatSmartDateTime(release.startedAt, t)
  return formatSmartDateTime(release.createdAt, t)
}

function ApplicationGatewayPanel({ applicationId, deploymentTargets, environments, projectId, ref, routes, servicePort }: {
  applicationId: string
  deploymentTargets: DeploymentTarget[]
  environments: Array<{ id: string, name: string }>
  projectId: string
  ref?: React.Ref<ApplicationPanelHandle>
  routes: GatewayRoute[]
  servicePort: number
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRoute, setEditingRoute] = useState<GatewayRoute | null>(null)
  const [routeToDelete, setRouteToDelete] = useState<GatewayRoute | null>(null)
  const form = useForm<RouteForm>({ defaultValues: routeDefaults, mode: 'onChange' })
  const deploymentTargetOptions = useMemo(() => deploymentTargets.map(target => ({
    id: target.id,
    label: gatewayDeploymentTargetLabel(target, environments, t),
  })), [deploymentTargets, environments, t])
  const saveRoute = useMutation({
    mutationFn: (values: RouteForm) => {
      const target = deploymentTargets.find(item => item.id === values.deploymentTargetId)
      const payload = {
        ...values,
        applicationId,
        environmentId: target?.environmentId ?? values.environmentId,
        servicePort: values.servicePort || servicePort,
      }
      return editingRoute ? api.updateGatewayRoute(projectId, editingRoute.id, payload) : api.createGatewayRoute(projectId, payload)
    },
    onSuccess: () => {
      toast.success(t(editingRoute ? 'gatewayRoutesPage.routeUpdated' : 'gatewayRoutesPage.routeCreated'))
      setDialogOpen(false)
      setEditingRoute(null)
      form.reset(routeDefaults)
      queryClient.invalidateQueries({ queryKey: ['gateway-routes', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteRoute = useMutation({
    mutationFn: (routeId: string) => api.deleteGatewayRoute(projectId, routeId),
    onSuccess: () => {
      toast.success(t('gatewayRoutesPage.routeDeleted'))
      setRouteToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['gateway-routes', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const checkDomain = useMutation({
    mutationFn: (host: string) => api.checkGatewayDomain(projectId, host),
    onSuccess: result => toast.success(result.available ? t('gatewayRoutesPage.domainAvailable') : t('gatewayRoutesPage.domainUnavailable')),
    onError: error => toast.error(error.message),
  })
  function openRouteDialog(route?: GatewayRoute) {
    setEditingRoute(route ?? null)
    const defaultTarget = deploymentTargets[0]
    const matchedTarget = route?.deploymentTargetId
      ? deploymentTargets.find(target => target.id === route.deploymentTargetId)
      : deploymentTargets.find(target => target.environmentId === route?.environmentId)
    form.reset(route
      ? { ...route, deploymentTargetId: route.deploymentTargetId || matchedTarget?.id || '', environmentId: matchedTarget?.environmentId ?? route.environmentId }
      : { ...routeDefaults, applicationId, deploymentTargetId: defaultTarget?.id ?? '', environmentId: defaultTarget?.environmentId ?? '', servicePort })
    setDialogOpen(true)
  }
  useImperativeHandle(ref, () => ({ openCreateDialog: () => openRouteDialog() }))
  return (
    <div className="grid gap-4">
      <DataList
        columns={[
          { key: 'host', header: t('gatewayRoutesPage.host'), render: item => item.host },
          { key: 'path', header: t('gatewayRoutesPage.path'), render: item => item.path },
          { key: 'tls', header: t('gatewayRoutesPage.tlsMode'), render: item => item.tlsMode },
          { key: 'status', header: t('common.status'), render: item => <StatusValueBadge value={item.status} /> },
          { key: 'actions', header: t('common.actions'), className: 'text-right whitespace-nowrap', render: item => (
            <div className="flex justify-end gap-2">
              <Button size="sm" variant="ghost" onClick={() => checkDomain.mutate(item.host)}>
                <SearchCheck className="size-4" />
                {t('gatewayRoutesPage.checkDomain')}
              </Button>
              <EditActionButton label={t('common.edit')} onClick={() => openRouteDialog(item)} />
              <Button size="sm" variant="ghost" onClick={() => setRouteToDelete(item)}>
                <Trash2 className="size-4" />
                {t('common.delete')}
              </Button>
            </div>
          ) },
        ]}
        emptyTitle={t('gatewayRoutesPage.emptyRoutes')}
        items={routes}
        rowKey={item => item.id}
      />
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingRoute ? t('gatewayRoutesPage.editRoute') : t('gatewayRoutesPage.createRoute')}</DialogTitle>
            <DialogDescription>{t('gatewayRoutesPage.routeDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={form.handleSubmit(values => saveRoute.mutate(values))}>
            <GatewayRouteFormFields
              deploymentTargetIdField={form.register('deploymentTargetId', { required: true })}
              deploymentTargets={deploymentTargetOptions}
              hostField={form.register('host')}
              pathField={form.register('path')}
              servicePortField={form.register('servicePort', { valueAsNumber: true })}
              tlsModeField={form.register('tlsMode')}
            />
            <DialogFooter><Button disabled={!form.formState.isValid || saveRoute.isPending} type="submit">{t('common.save')}</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <ConfirmDialog cancelText={t('common.cancel')} confirmText={t('common.delete')} description={t('gatewayRoutesPage.deleteRouteDescription')} open={Boolean(routeToDelete)} title={t('gatewayRoutesPage.deleteRouteTitle')} onConfirm={() => routeToDelete && deleteRoute.mutate(routeToDelete.id)} onOpenChange={open => !open && setRouteToDelete(null)} />
    </div>
  )
}

function firstReleaseReadyTarget(targets: DeploymentTarget[], runs: BuildRun[]) {
  const deployableRuns = latestDeployableBuildRuns(runs)
  return targets.find(target => deploymentTargetCanRelease(target, deployableRuns))
}

function deploymentTargetCanRelease(target: DeploymentTarget, deployableRuns: BuildRun[]) {
  if (!target.enabled)
    return false
  if (target.sourceType === 'image')
    return Boolean(target.imageRef?.trim())
  return deployableRuns.some(run => run.deploymentTargetId === target.id)
}

function redeployReleasePayload(target: DeploymentTarget, latestRelease?: Release): ReleaseForm | null {
  const imageRef = target.sourceType === 'image'
    ? (target.imageRef?.trim() || latestRelease?.imageRef?.trim() || '')
    : (latestRelease?.imageRef?.trim() || '')
  const buildRunId = target.sourceType === 'repository' ? (latestRelease?.buildRunId ?? '') : ''
  if (!imageRef)
    return null
  return {
    ...releaseDefaults,
    applicationId: target.applicationId,
    buildRunId,
    deploymentTargetId: target.id,
    environmentId: target.environmentId,
    imageRef,
    revision: (latestRelease?.revision ?? 0) + 1,
    status: 'pending',
    type: 'deploy',
  }
}

function deploymentTargetRuntimeChanged(current: DeploymentTarget, next: DeploymentTargetPayload) {
  const currentPayload = normalizeDeploymentTargetPayload({
    ...deploymentTargetDefaults,
    ...current,
    secretRefs: '',
  })
  const nextPayload = normalizeDeploymentTargetPayload(next)
  const fields: Array<keyof DeploymentTargetPayload> = [
    'environmentId',
    'sourceType',
    'runtimeConfigSetIds',
    'envVars',
    'configRefs',
    'configFiles',
    'dataRetentionEnabled',
    'dataCapacity',
    'dataMountPath',
    'dataVolumes',
  ]
  if (nextPayload.sourceType === 'image')
    fields.push('imageRef')
  if (String(nextPayload.secretRefs ?? '').trim() || String(nextPayload.secretFiles ?? '').trim())
    return true
  return fields.some(field => normalizedComparable(currentPayload[field]) !== normalizedComparable(nextPayload[field]))
}

function normalizedComparable(value: unknown) {
  if (typeof value === 'boolean')
    return value ? 'true' : 'false'
  if (typeof value === 'string')
    return value.trim()
  if (Array.isArray(value))
    return value.map(item => String(item).trim()).filter(Boolean).join(',')
  return String(value ?? '').trim()
}

function normalizeStringIds(value: unknown): string[] {
  if (Array.isArray(value))
    return value.map(item => String(item).trim()).filter(Boolean)
  if (typeof value !== 'string')
    return []
  const trimmed = value.trim()
  if (!trimmed)
    return []
  try {
    const parsed = JSON.parse(trimmed)
    if (Array.isArray(parsed))
      return parsed.map(item => String(item).trim()).filter(Boolean)
  }
  catch {
    return trimmed.split(',').map(item => item.trim()).filter(Boolean)
  }
  return []
}

function normalizeDeploymentTargetPayload(values: DeploymentTargetPayload): DeploymentTargetPayload {
  const enabled = normalizeBoolean(values.enabled, true)
  const autoDeploy = normalizeBoolean(values.autoDeploy, false)
  const requireApproval = normalizeBoolean(values.requireApproval, false)
  const buildHooksEnabled = normalizeBoolean(values.buildHooksEnabled, true)
  const dataRetentionEnabled = normalizeBoolean(values.dataRetentionEnabled, false)
  const dataVolumes = dataRetentionEnabled
    ? parseRuntimeDataVolumes(values.dataVolumes, values.dataMountPath || '/data', values.dataCapacity || '1Gi')
    : []
  const primaryDataVolume = dataVolumes[0]
  const sourceType = values.sourceType === 'image' ? 'image' : 'repository'
  return {
    ...values,
    sourceType,
    enabled,
    autoDeploy,
    requireApproval,
    buildHooksEnabled,
    dataRetentionEnabled,
    dataCapacity: dataRetentionEnabled ? (primaryDataVolume?.capacity?.trim() || '1Gi') : '',
    dataMountPath: dataRetentionEnabled ? (primaryDataVolume?.mountPath?.trim() || '/data') : '',
    dataVolumes: dataRetentionEnabled ? serializeRuntimeDataVolumes(dataVolumes) : '',
    repositoryBindingId: sourceType === 'repository' ? values.repositoryBindingId : '',
    targetRegistryId: sourceType === 'repository' ? values.targetRegistryId : '',
    targetImageRef: sourceType === 'repository' ? values.targetImageRef : '',
    imageRef: sourceType === 'image' ? values.imageRef : '',
    targetTag: values.targetTag || 'latest',
    buildVariableSetIds: normalizeStringIds(values.buildVariableSetIds),
    runtimeConfigSetIds: normalizeStringIds(values.runtimeConfigSetIds),
    configFiles: values.configFiles?.trim() ?? '',
    secretFiles: values.secretFiles?.trim() ?? '',
    buildHookBindings: values.buildHookBindings ?? [],
  }
}

function normalizeRuntimeConfigPayload(values: ProjectRuntimeConfigSetPayload): ProjectRuntimeConfigSetPayload {
  return {
    configFiles: values.configFiles?.trim() ?? '',
    enabled: Boolean(values.enabled),
    envVars: values.envVars?.trim() ?? '',
    name: values.name.trim(),
    secretFiles: values.secretFiles?.trim() ?? '',
    secretRefs: values.secretRefs?.trim() ?? '',
  }
}

function applyDockerfileBuildDefaults(form: UseFormReturn<DeploymentTargetPayload>, dockerfilePath: string, directories: string[]) {
  const normalizedDockerfile = dockerfilePath.trim()
  if (!normalizedDockerfile)
    return
  const buildContext = defaultBuildContextForDockerfile(normalizedDockerfile, directories)
  form.setValue('dockerfilePath', normalizedDockerfile, { shouldDirty: true, shouldValidate: true })
  form.setValue('buildContext', buildContext, { shouldDirty: true, shouldValidate: true })
  form.setValue('buildDirectory', buildContext === '.' ? '' : buildContext, { shouldDirty: true, shouldValidate: true })
}

function defaultBuildContextForDockerfile(dockerfilePath: string, directories: string[]) {
  const normalized = dockerfilePath.trim().replace(/^\/+/, '')
  const separatorIndex = normalized.lastIndexOf('/')
  if (separatorIndex < 0)
    return '.'
  const directory = normalized.slice(0, separatorIndex).trim()
  if (!directory)
    return '.'
  if (directories.length === 0 || directories.includes(directory))
    return directory
  const parent = directories
    .filter(option => option !== '.' && directory.startsWith(`${option}/`))
    .sort((left, right) => right.length - left.length)[0]
  return parent ?? directory
}

function normalizeBoolean(value: unknown, fallback: boolean) {
  if (typeof value === 'boolean')
    return value
  if (value === 'true')
    return true
  if (value === 'false')
    return false
  return fallback
}

function firstSelectableDeploymentTarget(configs: DeliveryConfigRow[]) {
  return configs.find(config => config.enabled) ?? configs[0]
}

function deploymentTargetImageRef(config?: DeliveryConfigRow) {
  if (!config?.targetRepository)
    return ''
  return `${config.targetRepository}:${config.targetTag || 'latest'}`
}

function registryInputPrefix(registry: ArtifactRegistry) {
  if (isDockerHubRegistry(registry))
    return ''
  const host = registryHost(registry.endpoint)
  return host ? `${host}/` : ''
}

function isDockerHubRegistry(registry: ArtifactRegistry) {
  return registry.provider === 'dockerhub' || registry.endpoint.includes('docker.io')
}

function registryHost(endpoint: string) {
  return endpoint.replace(/^https?:\/\//, '').replace(/\/.*$/, '')
}

function registryOptionLabel(registry: ArtifactRegistry) {
  return registry.namespace ? `${registry.name} / ${registry.namespace}` : registry.name
}

function branchOptions(values: Array<{ name: string }>, current?: string) {
  const options = values.map(branch => ({ value: branch.name, label: branch.name }))
  const normalized = current?.trim()
  if (normalized && !options.some(option => option.value === normalized))
    options.unshift({ value: normalized, label: normalized })
  return options
}

function defaultTargetImageRef(registry: ArtifactRegistry | undefined, projectSlug: string, appSlug: string) {
  const imageName = [slugSegment(projectSlug), slugSegment(appSlug)].filter(Boolean).join('-')
  if (!imageName)
    return ''
  const namespace = registry?.namespace?.trim().replace(/^\/+|\/+$/g, '')
  return `${namespace ? `${namespace}/` : ''}${imageName}:latest`
}

function slugSegment(value: string) {
  return value.trim().replace(/^\/+|\/+$/g, '').toLowerCase()
}
