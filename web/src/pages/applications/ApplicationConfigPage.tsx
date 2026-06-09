import type { ReactNode } from 'react'
import type { Application, ApplicationModule, ArtifactRegistry, BuildJob, BuildRun, DeploymentTarget, GatewayRoute, GitAccount, GitProvider, ProjectHookConfig, Release, RepositoryBinding } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { Activity, CalendarClock, CircleCheck, CircleX, Clock3, Eye, GitBranch, Globe2, GripVertical, LoaderCircle, MoreHorizontal, Package, Play, Plus, Rocket, RotateCcw, Save, ScrollText, Search, SearchCheck, Settings2, Square, Trash2, X } from 'lucide-react'
import { useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'
import { api, buildJobLogsStreamUrl } from '@/api/client'
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
import { SearchSelect } from '@/components/common/search-select'
import { SortableGrid } from '@/components/common/sortable-grid'
import { StatusValueBadge } from '@/components/common/status-badge'
import { TargetImageRefInput } from '@/components/common/target-image-ref-input'
import { formatElapsedDuration, formatSmartDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { TabsContent } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { WORKFLOW_STATUS_REFETCH_INTERVAL_MS } from '@/lib/polling'
import { APPLICATION_SLUG_MAX_LENGTH, DEPLOYMENT_SLUG_MAX_LENGTH } from '@/lib/slug-limits'

const schema = z.object({
  name: z.string().min(1, i18next.t('apps.nameRequired')),
  slug: z.string().min(1, i18next.t('apps.slugRequired')).max(APPLICATION_SLUG_MAX_LENGTH, i18next.t('apps.slugMaxLength', { count: APPLICATION_SLUG_MAX_LENGTH })).regex(/^[a-z0-9-]+$/, i18next.t('common.lowercaseSlugOnly')),
  icon: z.string().default('box'),
})

type ApplicationFormInput = z.input<typeof schema>
type ApplicationForm = z.output<typeof schema>
type TriggerForm = Partial<BuildRun>
type ReleaseForm = Omit<Release, 'id' | 'projectId' | 'createdBy' | 'createdAt' | 'rollbackFromId'>
type ModuleForm = Omit<ApplicationModule, 'id' | 'projectId' | 'applicationId' | 'createdBy' | 'createdAt'>
type DeploymentTargetForm = Omit<DeploymentTarget, 'id' | 'projectId' | 'applicationId' | 'createdBy' | 'createdAt'>
type DraftDeploymentTarget = DeploymentTargetForm & { id: string }
type ModuleTargetRow = DraftDeploymentTarget & { persisted?: DeploymentTarget }
type RouteForm = Omit<GatewayRoute, 'id' | 'projectId' | 'createdBy' | 'createdAt' | 'cnameName' | 'cnameTarget'> & { applicationSlug?: string, stage?: string }

const triggerDefaults: TriggerForm = { applicationId: '', moduleId: '', sourceBranch: '', targetImageRef: '', targetRegistryId: '', triggerType: 'manual' }
const releaseDefaults: ReleaseForm = { applicationId: '', moduleId: '', buildRunId: '', environmentId: '', imageRef: '', message: '', revision: 1, status: 'pending', type: 'deploy' }
const moduleDefaults: ModuleForm = { branchPattern: '', buildContext: '.', buildDirectory: '', buildHookBindings: [], buildHooksEnabled: true, buildLabels: '', buildProviderId: '', buildVariableSetIds: [], concurrencyPolicy: 'queue', dockerfilePath: 'Dockerfile', enabled: true, name: '', repositoryBindingId: '', slug: '', tagPattern: '', targetImageRef: '', targetRegistryId: '', targetRepository: '', targetTag: 'latest' }
const deploymentTargetDefaults: DeploymentTargetForm = { autoDeploy: false, branchPattern: '', moduleId: '', enabled: true, environmentId: '', name: '', requireApproval: false, tagPattern: '' }
const routeDefaults: RouteForm = { applicationId: '', applicationSlug: '', certificateStatus: 'disabled', dnsStatus: 'pending', environmentId: '', host: '', isDefault: false, path: '/', servicePort: 8080, stage: 'dev', status: 'pending', tlsMode: 'http-only' }
const buildRunStatusFilters: Array<BuildRun['status']> = ['queued', 'running', 'succeeded', 'failed', 'canceled', 'lost', 'timeout']
const buildRunEventFilters: Array<BuildRun['triggerType']> = ['manual', 'push', 'tag', 'webhook', 'api', 'retry']
const buildHookPhases: Array<Extract<ProjectHookConfig['phase'], 'preBuild' | 'postBuild'>> = ['preBuild', 'postBuild']
const APPLICATION_CONFIG_FORM_ID = 'application-config-form'

interface ApplicationPanelHandle {
  openCreateDialog: (environmentId?: string, moduleId?: string) => void
}
const buildTemplateVariableRows = [
  {
    descriptionKey: 'apps.templateVariables.fullSha',
    example: 'team/api:$' + '{{ github.sha }}',
    variable: '$' + '{{ github.sha }}',
  },
  {
    descriptionKey: 'apps.templateVariables.shortSha',
    example: 'team/api:{short_sha}',
    variable: '{short_sha}',
  },
  {
    descriptionKey: 'apps.templateVariables.refName',
    example: 'team/api:$' + '{{ github.ref_name }}',
    variable: '$' + '{{ github.ref_name }}',
  },
  {
    descriptionKey: 'apps.templateVariables.refType',
    example: 'team/api:$' + '{{ github.ref_type }}-{short_sha}',
    variable: '$' + '{{ github.ref_type }}',
  },
  {
    descriptionKey: 'apps.templateVariables.ref',
    example: 'team/api:$' + '{{ github.ref }}',
    variable: '$' + '{{ github.ref }}',
  },
] as const
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
  const deploymentsPanelRef = useRef<ApplicationPanelHandle>(null)
  const gatewayPanelRef = useRef<ApplicationPanelHandle>(null)
  const application = useQuery({
    queryKey: ['application', projectId, applicationId],
    queryFn: () => api.getApplication(projectId, applicationId),
    enabled: Boolean(projectId && applicationId),
  })
  const project = useQuery({ queryKey: ['project', projectId], queryFn: () => api.getProject(projectId), enabled: Boolean(projectId) })
  const repositoryBindings = useQuery({ queryKey: ['repository-bindings', projectId], queryFn: () => api.listRepositoryBindings(projectId), enabled: Boolean(projectId) })
  const registries = useQuery({ queryKey: ['registries', projectId], queryFn: () => api.listRegistries(projectId), enabled: Boolean(projectId) })
  const gitProviders = useQuery({ queryKey: ['git-providers'], queryFn: () => api.listGitProviders() })
  const gitAccounts = useQuery({ queryKey: ['git-accounts'], queryFn: () => api.listGitAccounts() })
  const modules = useQuery({ queryKey: ['application-modules', projectId, applicationId], queryFn: () => api.listApplicationModules(projectId, applicationId), enabled: Boolean(projectId && applicationId) })
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

  const binding = useMemo(() => (repositoryBindings.data ?? []).find(item => item.applicationId === applicationId), [applicationId, repositoryBindings.data])
  const appRepositoryBindings = useMemo(() => (repositoryBindings.data ?? []).filter(item => item.applicationId === applicationId), [applicationId, repositoryBindings.data])
  const appBuildRuns = useMemo(() => (buildRuns.data ?? []).filter(run => run.applicationId === applicationId), [applicationId, buildRuns.data])
  const appBuildRunIds = useMemo(() => new Set(appBuildRuns.map(run => run.id)), [appBuildRuns])
  const appBuildJobs = useMemo(() => (buildJobs.data ?? []).filter(job => appBuildRunIds.has(job.buildRunId)), [appBuildRunIds, buildJobs.data])
  const appReleases = useMemo(() => (releases.data ?? []).filter(release => release.applicationId === applicationId), [applicationId, releases.data])
  const appRoutes = useMemo(() => (routes.data ?? []).filter(route => route.applicationId === applicationId), [applicationId, routes.data])

  const updateForm = useForm<ApplicationFormInput, undefined, ApplicationForm>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { icon: 'box', name: '', slug: '' },
  })

  useEffect(() => {
    if (!application.data)
      return
    updateForm.reset({
      name: application.data.name,
      slug: application.data.slug,
      icon: application.data.icon ?? 'box',
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
      servicePort: application.data?.servicePort ?? 8080,
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
          { label: t('apps.modulesTab'), value: 'modules' },
          { label: t('builds'), value: 'builds' },
          { label: t('deployments'), value: 'deployments' },
          { label: t('gatewayRoutes'), value: 'gateway' },
        ]}
        tools={(
          <div className="flex items-center gap-2">
            {activeTab === 'deployments' && (
              <Button disabled={!appBuildRuns.some(run => run.status === 'succeeded' && Boolean(buildRunImageRef(run))) || !environments.data?.length} onClick={() => deploymentsPanelRef.current?.openCreateDialog()}>
                <Package size={16} />
                {t('deploymentsPage.createRelease')}
              </Button>
            )}
            {activeTab === 'gateway' && (
              <Button onClick={() => gatewayPanelRef.current?.openCreateDialog()}>
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
            deploymentTargets={deploymentTargets.data ?? []}
            modules={modules.data ?? []}
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
        <TabsContent value="modules">
          <div className="grid gap-4 xl:grid-cols-[minmax(0,44rem)_minmax(22rem,1fr)] xl:items-start">
            <ModuleManager
              applicationSlug={application.data?.slug ?? ''}
              applicationId={applicationId}
              modules={modules.data ?? []}
              deploymentTargets={deploymentTargets.data ?? []}
              environments={environments.data ?? []}
              gitAccounts={gitAccounts.data ?? []}
              gitProviders={gitProviders.data ?? []}
              projectId={projectId}
              registries={registries.data ?? []}
              repositoryBindings={appRepositoryBindings}
              routes={appRoutes}
              servicePort={application.data?.servicePort ?? 8080}
            />
            <BuildTemplateVariableTable />
          </div>
        </TabsContent>
        <TabsContent value="builds">
          <ApplicationBuildsPanel
            applicationId={applicationId}
            appSlug={application.data?.slug ?? ''}
            binding={binding}
            repositoryBindings={appRepositoryBindings}
            buildJobs={appBuildJobs}
            modules={modules.data ?? []}
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
            modules={modules.data ?? []}
            buildRuns={appBuildRuns}
            deploymentTargets={deploymentTargets.data ?? []}
            environments={environments.data ?? []}
            projectId={projectId}
            releases={appReleases}
          />
        </TabsContent>
        <TabsContent value="gateway">
          <ApplicationGatewayPanel
            ref={gatewayPanelRef}
            applicationId={applicationId}
            applicationSlug={application.data?.slug ?? ''}
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

function BuildTemplateVariableTable() {
  const { t } = useTranslation()
  return (
    <Card className="min-w-0 overflow-hidden p-0">
      <div className="border-b border-border px-3 py-2">
        <h3 className="text-sm font-medium">{t('apps.templateVariablesTitle')}</h3>
        <p className="mt-1 text-xs leading-5 text-muted-foreground">{t('apps.templateVariablesDescription')}</p>
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-44">{t('apps.templateVariablesColumn')}</TableHead>
              <TableHead className="min-w-56">{t('apps.templateVariableUsageColumn')}</TableHead>
              <TableHead className="min-w-52">{t('apps.templateVariableExampleColumn')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {buildTemplateVariableRows.map(row => (
              <TableRow key={row.variable}>
                <TableCell className="align-top">
                  <code className="rounded bg-background px-1.5 py-0.5 text-xs text-foreground">{row.variable}</code>
                </TableCell>
                <TableCell className="align-top text-sm text-muted-foreground">{t(row.descriptionKey)}</TableCell>
                <TableCell className="align-top">
                  <code className="break-all rounded bg-background px-1.5 py-0.5 text-xs text-foreground">{row.example}</code>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </Card>
  )
}

function ModuleManager({ applicationId, applicationSlug, modules, deploymentTargets, environments, gitAccounts, gitProviders, projectId, registries, repositoryBindings, routes, servicePort }: {
  applicationId: string
  applicationSlug: string
  modules: ApplicationModule[]
  deploymentTargets: DeploymentTarget[]
  environments: Array<{ id: string, name: string, stage?: string }>
  gitAccounts: GitAccount[]
  gitProviders: GitProvider[]
  projectId: string
  registries: ArtifactRegistry[]
  repositoryBindings: RepositoryBinding[]
  routes: GatewayRoute[]
  servicePort: number
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingConfig, setEditingConfig] = useState<ApplicationModule | null>(null)
  const [configToDelete, setConfigToDelete] = useState<ApplicationModule | null>(null)
  const [targetDialogOpen, setTargetDialogOpen] = useState(false)
  const [editingTarget, setEditingTarget] = useState<DeploymentTarget | null>(null)
  const [editingDraftTargetId, setEditingDraftTargetId] = useState<string | null>(null)
  const [targetToDelete, setTargetToDelete] = useState<DeploymentTarget | null>(null)
  const [draftTargets, setDraftTargets] = useState<DraftDeploymentTarget[]>([])
  const [gatewayEnabled, setGatewayEnabled] = useState(false)
  const [dockerfileSearch, setDockerfileSearch] = useState('')
  const [buildContextSearch, setBuildContextSearch] = useState('')
  const [selectedBuildHookIds, setSelectedBuildHookIds] = useState<string[]>([])
  const [targetImageRefEdited, setTargetImageRefEdited] = useState(false)
  const form = useForm<ModuleForm>({ defaultValues: moduleDefaults, mode: 'onChange' })
  const targetForm = useForm<DeploymentTargetForm>({ defaultValues: deploymentTargetDefaults, mode: 'onChange' })
  const gatewayForm = useForm<RouteForm>({ defaultValues: routeDefaults, mode: 'onChange' })
  const [repositoryPickerValue, setRepositoryPickerValue] = useState({
    cloneUrl: '',
    defaultBranch: 'main',
    gitAccountId: '',
    owner: '',
    repo: '',
  })
  const selectedRegistry = registries.find(registry => registry.id === form.watch('targetRegistryId'))
  const targetImagePrefix = selectedRegistry ? registryInputPrefix(selectedRegistry) : ''
  const watchedModuleSlug = form.watch('slug')
  const buildOptions = useQuery({
    queryKey: ['git-build-options', repositoryPickerValue.gitAccountId, repositoryPickerValue.owner, repositoryPickerValue.repo, repositoryPickerValue.defaultBranch],
    queryFn: () => api.getGitRepositoryBuildOptions(
      repositoryPickerValue.gitAccountId,
      repositoryPickerValue.owner,
      repositoryPickerValue.repo,
      repositoryPickerValue.defaultBranch,
    ),
    enabled: Boolean(dialogOpen && repositoryPickerValue.gitAccountId && repositoryPickerValue.owner && repositoryPickerValue.repo),
  })
  const projectHooks = useQuery({
    queryKey: ['project-hooks', projectId],
    queryFn: () => api.listProjectHooks(projectId),
    enabled: Boolean(dialogOpen && projectId),
  })
  const dockerfileOptions = useMemo(
    () => withCurrentOption(buildOptions.data?.dockerfiles ?? [], form.watch('dockerfilePath')).filter(option => option.value.toLowerCase().includes(dockerfileSearch.trim().toLowerCase())),
    [buildOptions.data?.dockerfiles, dockerfileSearch, form.watch('dockerfilePath')],
  )
  const buildContextOptions = useMemo(
    () => withCurrentOption(buildOptions.data?.directories ?? [], form.watch('buildContext')).filter(option => option.value.toLowerCase().includes(buildContextSearch.trim().toLowerCase())),
    [buildContextSearch, buildOptions.data?.directories, form.watch('buildContext')],
  )
  const configTargets = useMemo(
    () => editingConfig ? deploymentTargets.filter(target => target.moduleId === editingConfig.id) : [],
    [deploymentTargets, editingConfig],
  )
  const moduleTargetRows = useMemo<ModuleTargetRow[]>(
    () => editingConfig
      ? configTargets.map(target => ({ ...target, persisted: target }))
      : draftTargets,
    [configTargets, draftTargets, editingConfig],
  )
  const selectedGatewayEnvironmentId = gatewayForm.watch('environmentId')
  const selectedGatewayRoute = useMemo(
    () => routes.find(route => route.environmentId === selectedGatewayEnvironmentId),
    [routes, selectedGatewayEnvironmentId],
  )
  useEffect(() => {
    if (!dialogOpen || targetImageRefEdited)
      return
    const nextImageRef = defaultModuleTargetImageRef(selectedRegistry, applicationSlug, watchedModuleSlug)
    if (nextImageRef && form.getValues('targetImageRef') !== nextImageRef) {
      form.setValue('targetImageRef', nextImageRef, { shouldDirty: true, shouldValidate: true })
    }
  }, [applicationSlug, dialogOpen, form, selectedRegistry, targetImageRefEdited, watchedModuleSlug])
  useEffect(() => {
    if (!gatewayEnabled)
      return
    const route = routes.find(item => item.environmentId === selectedGatewayEnvironmentId)
    if (route) {
      gatewayForm.reset({ ...route, applicationSlug, stage: environments.find(environment => environment.id === route.environmentId)?.stage || 'dev' })
      return
    }
    gatewayForm.reset({
      ...routeDefaults,
      applicationId,
      applicationSlug,
      environmentId: selectedGatewayEnvironmentId || environments[0]?.id || '',
      servicePort,
      stage: environments.find(environment => environment.id === selectedGatewayEnvironmentId)?.stage || 'dev',
    })
  }, [applicationId, applicationSlug, environments, gatewayEnabled, gatewayForm, routes, selectedGatewayEnvironmentId, servicePort])
  const saveConfig = useMutation({
    mutationFn: async (values: ModuleForm) => {
      const repositoryBindingId = await ensureRepositoryBindingId()
      const payload = {
        ...values,
        buildProviderId: '',
        buildHookBindings: selectedBuildHookIds.map((hookConfigId, index) => ({ hookConfigId, runOrder: index + 1 })),
        buildVariableSetIds: [],
        repositoryBindingId,
      }
      const config = editingConfig
        ? await api.updateApplicationModule(projectId, applicationId, editingConfig.id, payload)
        : await api.createApplicationModule(projectId, applicationId, payload)
      if (!editingConfig) {
        for (const draftTarget of draftTargets) {
          const { id: _id, ...targetPayload } = draftTarget
          await api.createDeploymentTarget(projectId, applicationId, { ...targetPayload, moduleId: config.id })
        }
      }
      if (gatewayEnabled) {
        const gatewayValues = gatewayForm.getValues()
        const payload = {
          ...gatewayValues,
          applicationId,
          applicationSlug,
          servicePort: Number(gatewayValues.servicePort || servicePort),
        }
        const route = routes.find(item => item.environmentId === payload.environmentId)
        if (route)
          await api.updateGatewayRoute(projectId, route.id, payload)
        else
          await api.createGatewayRoute(projectId, payload)
      }
      return config
    },
    onSuccess: () => {
      toast.success(t(editingConfig ? 'apps.moduleUpdated' : 'apps.moduleCreated'))
      setDialogOpen(false)
      setEditingConfig(null)
      setDraftTargets([])
      form.reset(moduleDefaults)
      setSelectedBuildHookIds([])
      queryClient.invalidateQueries({ queryKey: ['application-modules', projectId, applicationId] })
      queryClient.invalidateQueries({ queryKey: ['deployment-targets', projectId, applicationId] })
      queryClient.invalidateQueries({ queryKey: ['gateway-routes', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  async function ensureRepositoryBindingId() {
    if (!repositoryPickerValue.gitAccountId || !repositoryPickerValue.owner || !repositoryPickerValue.repo)
      return ''
    const existing = repositoryBindings.find(binding =>
      binding.gitAccountId === repositoryPickerValue.gitAccountId
      && binding.owner === repositoryPickerValue.owner
      && binding.repo === repositoryPickerValue.repo,
    )
    if (existing)
      return existing.id
    const binding = await api.createRepositoryBinding(projectId, {
      applicationId,
      autoConfigureWebhook: true,
      cloneUrl: repositoryPickerValue.cloneUrl,
      defaultBranch: repositoryPickerValue.defaultBranch || 'main',
      gitAccountId: repositoryPickerValue.gitAccountId,
      owner: repositoryPickerValue.owner,
      repo: repositoryPickerValue.repo,
      webhookStatus: 'pending',
    })
    await queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
    return binding.id
  }
  const deleteConfig = useMutation({
    mutationFn: (configId: string) => api.deleteApplicationModule(projectId, applicationId, configId),
    onSuccess: () => {
      toast.success(t('apps.moduleDeleted'))
      setConfigToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['application-modules', projectId, applicationId] })
    },
    onError: error => toast.error(error.message),
  })
  const saveTarget = useMutation({
    mutationFn: (values: DeploymentTargetForm) => editingTarget
      ? api.updateDeploymentTarget(projectId, applicationId, editingTarget.id, values)
      : api.createDeploymentTarget(projectId, applicationId, { ...values, moduleId: editingConfig?.id ?? values.moduleId }),
    onSuccess: () => {
      toast.success(t(editingTarget ? 'deploymentsPage.targetUpdated' : 'deploymentsPage.targetCreated'))
      setTargetDialogOpen(false)
      setEditingTarget(null)
      targetForm.reset(deploymentTargetDefaults)
      queryClient.invalidateQueries({ queryKey: ['deployment-targets', projectId, applicationId] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteTarget = useMutation({
    mutationFn: (target: DeploymentTarget) => api.deleteDeploymentTarget(projectId, applicationId, target.id),
    onSuccess: () => {
      toast.success(t('deploymentsPage.targetDeleted'))
      setTargetToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['deployment-targets', projectId, applicationId] })
    },
    onError: error => toast.error(error.message),
  })
  function openConfigDialog(config?: ApplicationModule) {
    setEditingConfig(config ?? null)
    setDraftTargets([])
    const binding = config ? repositoryBindings.find(item => item.id === config.repositoryBindingId) : repositoryBindings[0]
    form.reset(config
      ? { ...config, buildVariableSetIds: [], targetImageRef: moduleTargetImageRef(config) }
      : { ...moduleDefaults, repositoryBindingId: binding?.id ?? '' })
    setTargetImageRefEdited(Boolean(config?.targetRepository))
    setSelectedBuildHookIds(config?.buildHookBindings?.map(binding => binding.hookConfigId) ?? [])
    setRepositoryPickerValue({
      cloneUrl: binding?.cloneUrl ?? '',
      defaultBranch: binding?.defaultBranch ?? 'main',
      gitAccountId: binding?.gitAccountId ?? '',
      owner: binding?.owner ?? '',
      repo: binding?.repo ?? '',
    })
    setDockerfileSearch('')
    setBuildContextSearch('')
    const defaultEnvironmentId = config
      ? deploymentTargets.find(target => target.moduleId === config.id)?.environmentId || environments[0]?.id || ''
      : environments[0]?.id || ''
    const route = routes.find(item => item.environmentId === defaultEnvironmentId)
    setGatewayEnabled(Boolean(route))
    gatewayForm.reset(route
      ? { ...route, applicationSlug, stage: environments.find(environment => environment.id === route.environmentId)?.stage || 'dev' }
      : { ...routeDefaults, applicationId, applicationSlug, environmentId: defaultEnvironmentId, servicePort })
    setDialogOpen(true)
  }
  function openTargetDialog(target?: ModuleTargetRow) {
    setEditingTarget(target?.persisted ?? null)
    setEditingDraftTargetId(target && !target.persisted ? target.id : null)
    targetForm.reset(target ?? { ...deploymentTargetDefaults, moduleId: editingConfig?.id ?? '', environmentId: environments[0]?.id ?? '' })
    setTargetDialogOpen(true)
  }
  function saveDeploymentTarget(values: DeploymentTargetForm) {
    if (editingConfig) {
      saveTarget.mutate(values)
      return
    }
    const target: DraftDeploymentTarget = {
      ...deploymentTargetDefaults,
      ...values,
      id: editingDraftTargetId ?? crypto.randomUUID(),
      moduleId: '',
    }
    setDraftTargets(current => editingDraftTargetId
      ? current.map(item => item.id === editingDraftTargetId ? target : item)
      : [...current, target])
    setTargetDialogOpen(false)
    setEditingDraftTargetId(null)
    targetForm.reset(deploymentTargetDefaults)
  }
  function deleteModuleTarget(row: ModuleTargetRow) {
    if (row.persisted) {
      setTargetToDelete(row.persisted)
      return
    }
    setDraftTargets(current => current.filter(item => item.id !== row.id))
  }
  return (
    <Card className="min-w-0 overflow-hidden p-0">
      <div className="flex items-center justify-between gap-3 border-b border-border px-3 py-2">
        <div className="min-w-0">
          <h3 className="text-sm font-medium">{t('apps.modulesTitle')}</h3>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">{t('apps.modulesDescription')}</p>
        </div>
        <Button size="sm" type="button" onClick={() => openConfigDialog()}>
          <Plus className="size-4" />
          {t('apps.createModule')}
        </Button>
      </div>
      <DataList
        columns={[
          { key: 'name', header: t('common.name'), className: 'w-32 max-w-32 px-4 py-3 align-middle', render: item => <span className="block max-w-24 truncate whitespace-nowrap" title={item.name}>{item.name}</span> },
          { key: 'slug', header: t('common.slug'), className: 'w-24 max-w-24 px-4 py-3 align-middle', render: item => <code className="block max-w-16 truncate whitespace-nowrap rounded bg-background px-2 py-1 text-xs" title={item.slug}>{item.slug}</code> },
          { key: 'target', header: t('buildsPage.targetImage'), className: 'w-60 max-w-60 px-4 py-3 align-middle', render: item => <span className="block max-w-52 truncate whitespace-nowrap font-mono text-sm" title={moduleTargetImageRef(item)}>{moduleTargetImageRef(item) || '-'}</span> },
          { key: 'concurrency', header: t('apps.buildConcurrencyPolicy'), className: 'w-28 whitespace-nowrap px-4 py-3 align-middle', render: item => <span className="block max-w-24 truncate" title={t(`apps.buildConcurrencyPolicies.${item.concurrencyPolicy || 'queue'}`)}>{t(`apps.buildConcurrencyPolicies.${item.concurrencyPolicy || 'queue'}`)}</span> },
          { key: 'status', header: t('common.status'), className: 'w-28 whitespace-nowrap px-4 py-3 align-middle', render: item => <StatusValueBadge value={item.enabled ? 'enabled' : 'disabled'} /> },
          { key: 'actions', header: t('common.actions'), className: 'w-44 whitespace-nowrap px-4 py-3 text-right align-middle', render: item => (
            <div className="flex justify-end gap-2">
              <EditActionButton label={t('common.edit')} onClick={() => openConfigDialog(item)} />
              <Button size="sm" variant="ghost" onClick={() => setConfigToDelete(item)}>
                <Trash2 className="size-4" />
                {t('common.delete')}
              </Button>
            </div>
          ) },
        ]}
        emptyTitle={t('apps.emptyModules')}
        items={modules}
        rowKey={item => item.id}
        variant="plain"
      />
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="flex max-h-[88vh] max-w-5xl flex-col overflow-hidden p-0">
          <DialogHeader>
            <div className="px-6 pt-6">
              <DialogTitle>{editingConfig ? t('apps.editModule') : t('apps.createModule')}</DialogTitle>
              <DialogDescription>{t('apps.moduleDialogDescription')}</DialogDescription>
            </div>
          </DialogHeader>
          <form className="flex min-h-0 flex-1 flex-col" onSubmit={form.handleSubmit(values => saveConfig.mutate(values))}>
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">
              <div className="grid gap-4">
                <ModuleFormSection
                  description={t('apps.moduleBasicSectionDescription')}
                  icon={<Settings2 className="size-4" />}
                  title={t('apps.moduleBasicSection')}
                >
                  <div className="grid gap-3 md:grid-cols-2">
                    <Field label={t('common.name')} required><Input {...form.register('name', { required: true })} /></Field>
                    <Field hint={t('apps.moduleSlugHint', { count: DEPLOYMENT_SLUG_MAX_LENGTH })} label={t('common.slug')} required><Input {...form.register('slug', { required: true, maxLength: DEPLOYMENT_SLUG_MAX_LENGTH })} maxLength={DEPLOYMENT_SLUG_MAX_LENGTH} /></Field>
                  </div>
                  <div className="flex flex-wrap gap-4 text-sm text-foreground">
                    <label className="flex items-center gap-2">
                      <input className="size-4 accent-primary" type="checkbox" {...form.register('enabled')} />
                      {t('common.enabled')}
                    </label>
                  </div>
                </ModuleFormSection>

                <ModuleFormSection
                  description={t('apps.moduleSourceSectionDescription')}
                  icon={<GitBranch className="size-4" />}
                  title={t('apps.moduleSourceSection')}
                >
                  <GitRepositoryPicker
                    accounts={gitAccounts}
                    providers={gitProviders}
                    value={repositoryPickerValue}
                    onChange={(next) => {
                      setRepositoryPickerValue(next)
                      const existing = repositoryBindings.find(binding =>
                        binding.gitAccountId === next.gitAccountId
                        && binding.owner === next.owner
                        && binding.repo === next.repo,
                      )
                      form.setValue('repositoryBindingId', existing?.id ?? '', { shouldDirty: true, shouldValidate: true })
                      form.setValue('dockerfilePath', 'Dockerfile', { shouldDirty: true, shouldValidate: true })
                      form.setValue('buildContext', '.', { shouldDirty: true, shouldValidate: true })
                      form.setValue('buildDirectory', '.', { shouldDirty: true, shouldValidate: true })
                      setTargetImageRefEdited(false)
                      setDockerfileSearch('')
                      setBuildContextSearch('')
                    }}
                  />
                  <div className="grid gap-3 md:grid-cols-3">
                    <Field hint={t('buildsPage.buildContextLookupHint')} label={t('buildsPage.dockerfilePath')}>
                      <SearchSelect
                        emptyLabel={repositoryPickerValue.owner && repositoryPickerValue.repo ? t('common.noOptions') : t('buildsPage.repositoryBindingRequired')}
                        loading={buildOptions.isFetching}
                        options={dockerfileOptions}
                        placeholder={t('buildsPage.dockerfilePath')}
                        search={dockerfileSearch}
                        value={form.watch('dockerfilePath') || ''}
                        onSearchChange={(value) => {
                          setDockerfileSearch(value)
                          form.setValue('dockerfilePath', value, { shouldDirty: true, shouldValidate: true })
                        }}
                        onValueChange={(value) => {
                          form.setValue('dockerfilePath', value, { shouldDirty: true, shouldValidate: true })
                          const nextContext = dockerfileDirectory(value)
                          if (nextContext) {
                            form.setValue('buildContext', nextContext, { shouldDirty: true, shouldValidate: true })
                            form.setValue('buildDirectory', nextContext, { shouldDirty: true, shouldValidate: true })
                            setBuildContextSearch(nextContext)
                          }
                        }}
                      />
                    </Field>
                    <Field hint={t('buildsPage.buildContextLookupHint')} label={t('buildsPage.buildContext')}>
                      <SearchSelect
                        emptyLabel={repositoryPickerValue.owner && repositoryPickerValue.repo ? t('common.noOptions') : t('buildsPage.repositoryBindingRequired')}
                        loading={buildOptions.isFetching}
                        options={buildContextOptions}
                        placeholder={t('buildsPage.buildContextPlaceholder')}
                        search={buildContextSearch}
                        value={form.watch('buildContext') || ''}
                        onSearchChange={(value) => {
                          setBuildContextSearch(value)
                          form.setValue('buildContext', value || '.', { shouldDirty: true, shouldValidate: true })
                        }}
                        onValueChange={value => form.setValue('buildContext', value || '.', { shouldDirty: true, shouldValidate: true })}
                      />
                    </Field>
                    <Field hint={t('buildsPage.buildDirectoryHint')} label={t('buildsPage.buildDirectory')}>
                      <Input {...form.register('buildDirectory')} placeholder={t('buildsPage.buildDirectoryPlaceholder')} />
                    </Field>
                  </div>
                </ModuleFormSection>

                <ModuleFormSection
                  description={t('apps.moduleArtifactSectionDescription')}
                  icon={<Package className="size-4" />}
                  title={t('apps.moduleArtifactSection')}
                >
                  <div className="grid gap-3 md:grid-cols-[minmax(14rem,0.8fr)_minmax(0,1.2fr)]">
                    <Field label={t('buildsPage.targetRegistry')}>
                      <Select {...form.register('targetRegistryId')}>
                        <option value="">{t('common.select')}</option>
                        {registries.map(registry => <option key={registry.id} value={registry.id}>{registryOptionLabel(registry)}</option>)}
                      </Select>
                    </Field>
                    <Field hint={t('buildsPage.targetImageRefHint')} label={t('buildsPage.targetImageRef')}>
                      <TargetImageRefInput
                        placeholder={t('buildsPage.targetImageRefPlaceholder')}
                        prefix={targetImagePrefix}
                        register={form.register('targetImageRef', { onChange: () => setTargetImageRefEdited(true) })}
                      />
                    </Field>
                  </div>
                </ModuleFormSection>

                <ModuleFormSection
                  description={t('apps.moduleScheduleSectionDescription')}
                  icon={<Rocket className="size-4" />}
                  title={t('apps.moduleScheduleSection')}
                >
                  <div className="grid gap-3 md:grid-cols-2">
                    <Field hint={t('apps.buildConcurrencyPolicyHint')} label={t('apps.buildConcurrencyPolicy')}>
                      <Select {...form.register('concurrencyPolicy')}>
                        <option value="queue">{t('apps.buildConcurrencyPolicies.queue')}</option>
                        <option value="parallel">{t('apps.buildConcurrencyPolicies.parallel')}</option>
                      </Select>
                    </Field>
                    <Field label={t('apps.buildLabels')}><Input {...form.register('buildLabels')} placeholder={t('apps.buildLabelsPlaceholder')} /></Field>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    <Field hint={t('deploymentsPage.branchPatternHint')} label={t('deploymentsPage.branchPattern')}><Input {...form.register('branchPattern')} placeholder="main,release-*" /></Field>
                    <Field hint={t('deploymentsPage.tagPatternHint')} label={t('deploymentsPage.tagPattern')}><Input {...form.register('tagPattern')} placeholder="v*" /></Field>
                  </div>
                </ModuleFormSection>

                <ModuleFormSection
                  description={t('apps.moduleBuildHooksSectionDescription')}
                  icon={<ScrollText className="size-4" />}
                  title={t('apps.moduleBuildHooksSection')}
                >
                  <ModuleBuildHookSelector
                    hooks={projectHooks.data ?? []}
                    loading={projectHooks.isFetching}
                    selectedIds={selectedBuildHookIds}
                    onSelectedIdsChange={setSelectedBuildHookIds}
                  />
                </ModuleFormSection>

                <ModuleFormSection
                  description={editingConfig ? t('deploymentsPage.deploymentConfigsDescription') : t('deploymentsPage.saveModuleBeforeDeploymentConfig')}
                  icon={<Rocket className="size-4" />}
                  title={t('deploymentsPage.deploymentConfigs')}
                  actions={(
                    <Button disabled={!environments.length} size="sm" type="button" variant="secondary" onClick={() => openTargetDialog()}>
                      <Plus className="size-4" />
                      {t('deploymentsPage.createDeploymentConfig')}
                    </Button>
                  )}
                >
                  <DataList
                    columns={[
                      { key: 'name', header: t('common.name'), render: item => item.name || '-' },
                      { key: 'environment', header: t('deploymentsPage.environment'), render: item => releaseEnvironmentLabel(environments.find(environment => environment.id === item.environmentId), item.environmentId, t) },
                      { key: 'auto', header: t('deploymentsPage.autoDeploy'), render: item => <StatusValueBadge value={item.autoDeploy ? 'enabled' : 'disabled'} /> },
                      { key: 'status', header: t('common.status'), render: item => <StatusValueBadge value={item.enabled ? 'enabled' : 'disabled'} /> },
                      { key: 'actions', header: t('common.actions'), className: 'text-right whitespace-nowrap', render: item => (
                        <div className="flex justify-end gap-2">
                          <EditActionButton label={t('common.edit')} size="sm" onClick={() => openTargetDialog(item)} />
                          <Button size="sm" variant="ghost" onClick={() => deleteModuleTarget(item)}>
                            <Trash2 className="size-4" />
                            {t('common.delete')}
                          </Button>
                        </div>
                      ) },
                    ]}
                    emptyDescription={t('deploymentsPage.emptyDeploymentConfigsDescription')}
                    emptyTitle={t('deploymentsPage.emptyDeploymentConfigs')}
                    items={moduleTargetRows}
                    rowKey={item => item.id}
                    variant="plain"
                  />
                </ModuleFormSection>

                <ModuleFormSection
                  description={t('gatewayRoutesPage.optionalGatewayConfigDescription')}
                  icon={<Globe2 className="size-4" />}
                  title={t('gatewayRoutesPage.optionalGatewayConfig')}
                  actions={(
                    <label className="flex shrink-0 items-center gap-2 text-sm text-foreground">
                      <input
                        checked={gatewayEnabled}
                        className="size-4 accent-primary"
                        type="checkbox"
                        onChange={event => setGatewayEnabled(event.target.checked)}
                      />
                      {t('gatewayRoutesPage.enableGatewayConfig')}
                    </label>
                  )}
                >
                  {gatewayEnabled && (
                    <>
                      {selectedGatewayRoute && (
                        <div className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
                          {t('gatewayRoutesPage.gatewayRouteWillUpdate')}
                        </div>
                      )}
                      <div className="grid gap-3 md:grid-cols-2">
                        <GatewayRouteFormFields
                          environmentIdField={gatewayForm.register('environmentId')}
                          environments={environments}
                          hostField={gatewayForm.register('host')}
                          pathField={gatewayForm.register('path')}
                          servicePortField={gatewayForm.register('servicePort', { valueAsNumber: true })}
                          stageField={gatewayForm.register('stage')}
                          tlsModeField={gatewayForm.register('tlsMode')}
                        />
                      </div>
                    </>
                  )}
                </ModuleFormSection>
              </div>
            </div>
            <DialogFooter className="border-t border-border bg-background px-6 py-4">
              <Button disabled={!form.formState.isValid || saveConfig.isPending} type="submit">{t('common.save')}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog open={targetDialogOpen} onOpenChange={setTargetDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingTarget ? t('deploymentsPage.editDeploymentConfig') : t('deploymentsPage.createDeploymentConfig')}</DialogTitle>
            <DialogDescription>{t('deploymentsPage.deploymentConfigDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={targetForm.handleSubmit(saveDeploymentTarget)}>
            <Field error={targetForm.formState.errors.name?.message} hint={t('deploymentsPage.deploymentConfigNameHint')} label={t('common.name')}>
              <Input {...targetForm.register('name')} />
            </Field>
            <Field label={t('deploymentsPage.environment')} required>
              <Select {...targetForm.register('environmentId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {environments.map(environment => <option key={environment.id} value={environment.id}>{releaseEnvironmentLabel(environment, environment.id, t)}</option>)}
              </Select>
            </Field>
            <div className="flex flex-wrap gap-4 text-sm text-foreground">
              <label className="flex items-center gap-2">
                <input className="size-4 accent-primary" type="checkbox" {...targetForm.register('autoDeploy')} />
                {t('deploymentsPage.autoDeploy')}
              </label>
              <label className="flex items-center gap-2">
                <input className="size-4 accent-primary" type="checkbox" {...targetForm.register('requireApproval')} />
                {t('deploymentsPage.requireApproval')}
              </label>
              <label className="flex items-center gap-2">
                <input className="size-4 accent-primary" type="checkbox" {...targetForm.register('enabled')} />
                {t('common.enabled')}
              </label>
            </div>
            <DialogFooter><Button disabled={!targetForm.formState.isValid || (Boolean(editingConfig) && saveTarget.isPending)} type="submit">{t('common.save')}</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('apps.deleteModuleDescription')}
        open={Boolean(configToDelete)}
        title={t('apps.deleteModuleTitle')}
        onConfirm={() => configToDelete && deleteConfig.mutate(configToDelete.id)}
        onOpenChange={open => !open && setConfigToDelete(null)}
      />
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('deploymentsPage.deleteDeploymentConfigDescription')}
        open={Boolean(targetToDelete)}
        title={t('deploymentsPage.deleteDeploymentConfigTitle')}
        onConfirm={() => targetToDelete && deleteTarget.mutate(targetToDelete)}
        onOpenChange={open => !open && setTargetToDelete(null)}
      />
    </Card>
  )
}

function ApplicationOverviewPanel({ app, buildRuns, deploymentTargets, modules, releases, routes }: {
  app?: Application
  buildRuns: BuildRun[]
  deploymentTargets: DeploymentTarget[]
  modules: ApplicationModule[]
  releases: Release[]
  routes: GatewayRoute[]
}) {
  const { t } = useTranslation()
  const enabledModules = modules.filter(module => module.enabled).length
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
          label={t('apps.moduleHealth')}
          meta={t('apps.enabledTotal', { enabled: enabledModules, total: modules.length })}
          value={String(modules.length)}
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
            <OverviewItem label={t('apps.modulesTitle')} value={t('apps.enabledTotal', { enabled: enabledModules, total: modules.length })} />
            <OverviewItem label={t('deploymentsPage.deploymentConfigs')} value={t('apps.enabledTotal', { enabled: enabledTargets, total: deploymentTargets.length })} />
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

function ModuleFormSection({ actions, children, description, icon, title }: {
  actions?: ReactNode
  children: ReactNode
  description: string
  icon: ReactNode
  title: string
}) {
  return (
    <section className="min-w-0 overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border bg-muted/35 px-4 py-3">
        <div className="flex min-w-0 items-start gap-3">
          <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-background text-muted-foreground shadow-sm">
            {icon}
          </span>
          <div className="min-w-0">
            <h4 className="text-sm font-semibold">{title}</h4>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">{description}</p>
          </div>
        </div>
        {actions}
      </div>
      <div className="grid gap-4 p-4">
        {children}
      </div>
    </section>
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
    releases.filter(release => release.environmentId === target.environmentId && release.moduleId === target.moduleId),
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

function ApplicationBuildsPanel({ applicationId, appSlug, binding, modules, buildJobs, buildRuns, projectId, projectSlug, registries, repositoryBindings }: {
  applicationId: string
  appSlug: string
  binding?: { defaultBranch: string, gitAccountId: string, owner: string, repo: string }
  repositoryBindings: RepositoryBinding[]
  modules: ApplicationModule[]
  buildJobs: BuildJob[]
  buildRuns: BuildRun[]
  projectId: string
  projectSlug: string
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
  const selectedModule = modules.find(config => config.id === form.watch('moduleId')) ?? firstSelectableModule(modules)
  const selectedBinding = repositoryBindings.find(item => item.id === selectedModule?.repositoryBindingId) ?? binding
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
    const defaultConfig = firstSelectableModule(modules)
    form.reset({
      ...triggerDefaults,
      applicationId,
      moduleId: defaultConfig?.id ?? '',
      sourceBranch: (repositoryBindings.find(item => item.id === defaultConfig?.repositoryBindingId) ?? binding)?.defaultBranch || 'main',
      targetImageRef: moduleTargetImageRef(defaultConfig) || defaultTargetImageRef(undefined, projectSlug, appSlug),
      targetRegistryId: defaultConfig?.targetRegistryId ?? '',
    })
  }, [applicationId, appSlug, binding, modules, dialogOpen, form, projectSlug, repositoryBindings])

  useEffect(() => {
    if (!dialogOpen || !selectedModule)
      return
    const nextBinding = repositoryBindings.find(item => item.id === selectedModule.repositoryBindingId) ?? binding
    if (nextBinding)
      form.setValue('sourceBranch', nextBinding.defaultBranch || 'main', { shouldDirty: true, shouldValidate: true })
    if (selectedModule.targetRegistryId)
      form.setValue('targetRegistryId', selectedModule.targetRegistryId, { shouldDirty: true, shouldValidate: true })
    const configTargetImageRef = moduleTargetImageRef(selectedModule)
    if (configTargetImageRef)
      form.setValue('targetImageRef', configTargetImageRef, { shouldDirty: true, shouldValidate: true })
  }, [binding, dialogOpen, form, repositoryBindings, selectedModule])

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
                  <Button disabled={!repositoryBindings.length && !binding} onClick={() => setDialogOpen(true)}>
                    <Play className="size-4" />
                    {t('buildsPage.triggerBuild')}
                  </Button>
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
                        const config = modules.find(config => config.id === run.moduleId)
                        const rowBinding = repositoryBindings.find(binding => binding.id === config?.repositoryBindingId) ?? binding
                        if (!rowBinding)
                          return null
                        return (
                          <BuildRunRow
                            key={run.id}
                            binding={rowBinding}
                            moduleName={config?.name}
                            jobs={jobs}
                            latestJob={latestJob}
                            run={run}
                            canceling={cancelBuild.isPending}
                            retrying={retryBuild.isPending}
                            onCancel={() => cancelBuild.mutate(run.id)}
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
            <Field hint={t('buildsPage.moduleHint')} label={t('buildsPage.module')} required>
              <Select {...form.register('moduleId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {modules.map(config => <option key={config.id} value={config.id}>{config.name}</option>)}
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
              <Input readOnly value={selectedModule?.dockerfilePath || 'Dockerfile'} />
            </Field>
            <Field hint={t('buildsPage.inheritedModuleHint')} label={t('buildsPage.buildContext')}>
              <Input readOnly value={selectedModule?.buildContext || '.'} />
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

function BuildRunRow({ binding, moduleName, canceling, jobs, latestJob, onCancel, onOpenLogs, onRetry, retrying, run }: {
  binding: { cloneUrl?: string, defaultBranch: string, gitAccountId: string, owner: string, repo: string }
  moduleName?: string
  canceling: boolean
  jobs: BuildJob[]
  latestJob?: BuildJob
  onCancel: () => void
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
            <h3 className="truncate text-sm font-semibold text-foreground" title={buildRunTitle(run, t, moduleName)}>
              {buildRunTitle(run, t, moduleName)}
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

function buildRunTitle(run: BuildRun, t: ReturnType<typeof useTranslation>['t'], moduleName?: string) {
  let title = t('buildsPage.runTitleManual')
  if (run.triggerType === 'webhook' || run.triggerType === 'push')
    title = t('buildsPage.runTitlePush')
  else if (run.triggerType === 'tag')
    title = t('buildsPage.runTitleTag')
  else if (run.triggerType === 'api')
    title = t('buildsPage.runTitleApi')
  else if (run.triggerType === 'retry')
    title = t('buildsPage.runTitleRetry')
  return moduleName ? t('buildsPage.runTitleWithConfig', { config: moduleName, title }) : title
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

function ApplicationDeploymentsPanel({ applicationId, modules, buildRuns, deploymentTargets, environments, projectId, ref, releases }: {
  applicationId: string
  modules: ApplicationModule[]
  buildRuns: BuildRun[]
  deploymentTargets: DeploymentTarget[]
  environments: Array<{ id: string, name: string, stage?: string }>
  projectId: string
  ref?: React.Ref<ApplicationPanelHandle>
  releases: Release[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [logRelease, setLogRelease] = useState<Release | null>(null)
  const [targetToDelete, setTargetToDelete] = useState<DeploymentTarget | null>(null)
  const form = useForm<ReleaseForm>({ defaultValues: releaseDefaults, mode: 'onChange' })
  const buildRunMap = useMemo(() => Object.fromEntries(buildRuns.map(run => [run.id, run])), [buildRuns])
  const latestReleaseByTarget = useMemo(() => {
    const output: Record<string, Release> = {}
    for (const release of releases) {
      const key = deploymentReleaseKey(release.environmentId, release.moduleId)
      const existing = output[key]
      if (!existing || new Date(release.createdAt).getTime() > new Date(existing.createdAt).getTime())
        output[key] = release
    }
    return output
  }, [releases])
  const deployableBuildRuns = useMemo(() => latestDeployableBuildRuns(buildRuns), [buildRuns])
  const selectedModuleId = form.watch('moduleId')
  const selectableBuildRuns = useMemo(
    () => selectedModuleId ? deployableBuildRuns.filter(run => run.moduleId === selectedModuleId) : deployableBuildRuns,
    [deployableBuildRuns, selectedModuleId],
  )
  const deploymentRows = useMemo(() => deploymentTargets.map(target => ({
    module: modules.find(config => config.id === target.moduleId),
    environment: environments.find(environment => environment.id === target.environmentId),
    release: latestReleaseByTarget[deploymentReleaseKey(target.environmentId, target.moduleId)],
    target,
  })), [modules, deploymentTargets, environments, latestReleaseByTarget])
  const selectedBuildRun = buildRunMap[form.watch('buildRunId')]
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
  const openCreateDialog = (environmentId = '', moduleId = '') => {
    const matchedRun = moduleId ? deployableBuildRuns.find(run => run.moduleId === moduleId) : undefined
    form.reset({
      ...releaseDefaults,
      applicationId: matchedRun?.applicationId ?? '',
      moduleId,
      buildRunId: matchedRun?.id ?? '',
      environmentId,
      imageRef: matchedRun ? buildRunImageRef(matchedRun) : '',
    })
    setDialogOpen(true)
  }
  useImperativeHandle(ref, () => ({ openCreateDialog }))
  useEffect(() => {
    if (!selectedBuildRun)
      return
    form.setValue('moduleId', selectedBuildRun.moduleId || '', { shouldDirty: true, shouldValidate: true })
    form.setValue('applicationId', selectedBuildRun.applicationId, { shouldDirty: true, shouldValidate: true })
    form.setValue('imageRef', buildRunImageRef(selectedBuildRun), { shouldDirty: true, shouldValidate: true })
  }, [form, selectedBuildRun])
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
  return (
    <div className="grid gap-4">
      <DataList
        columns={[
          { key: 'name', header: t('common.name'), className: 'w-28 px-4 py-3 align-middle', render: item => <span className="block max-w-24 truncate" title={item.target.name}>{item.target.name}</span> },
          { key: 'module', header: t('buildsPage.module'), className: 'w-32 px-4 py-3 align-middle', render: item => <span className="block max-w-28 truncate" title={item.module?.name ?? item.target.moduleId}>{item.module?.name ?? item.target.moduleId}</span> },
          { key: 'environment', header: t('deploymentsPage.environment'), className: 'w-28 px-4 py-3 align-middle', render: item => <span className="block max-w-24 truncate" title={releaseEnvironmentLabel(item.environment, item.target.environmentId, t)}>{releaseEnvironmentLabel(item.environment, item.target.environmentId, t)}</span> },
          { key: 'auto', header: t('deploymentsPage.autoDeploy'), className: 'w-24 whitespace-nowrap px-4 py-3 align-middle', render: item => <StatusValueBadge value={item.target.autoDeploy ? 'enabled' : 'disabled'} /> },
          { key: 'revision', header: t('deploymentsPage.revision'), className: 'w-16 whitespace-nowrap px-4 py-3 align-middle', render: item => item.release ? `#${item.release.revision}` : '-' },
          { key: 'image', header: t('deploymentsPage.image'), className: 'w-52 px-4 py-3 align-middle', render: item => item.release ? <CopyableTruncatedText className="max-w-44 rounded bg-background px-2 py-1 font-mono text-xs" display={shortImageRef(item.release.imageRef)} value={item.release.imageRef} onCopy={copyDeploymentText} /> : '-' },
          { key: 'status', header: t('common.status'), className: 'w-20 whitespace-nowrap px-4 py-3 align-middle', render: item => item.release ? <StatusValueBadge labelKeyPrefix="buildsPage.statuses" value={item.release.status} /> : <StatusValueBadge label={t('deploymentsPage.notDeployed')} value="pending" /> },
          { key: 'message', header: t('deploymentsPage.rolloutMessage'), className: 'w-48 px-4 py-3 align-middle', render: item => <CopyableTruncatedText className="max-w-40 text-sm text-muted-foreground" display={compactReleaseMessage(item.release?.message)} value={item.release?.message} onCopy={copyDeploymentText} /> },
          { key: 'time', header: t('deploymentsPage.releaseTime'), className: 'w-24 px-4 py-3 align-middle', render: item => item.release ? formatReleaseTime(item.release, t) : '-' },
          { key: 'actions', header: t('common.actions'), className: 'text-right whitespace-nowrap', render: item => (
            <div className="flex justify-end gap-2">
              <Button disabled={!deployableBuildRuns.some(run => run.moduleId === item.target.moduleId) || createRelease.isPending} size="sm" variant="ghost" onClick={() => openCreateDialog(item.target.environmentId, item.target.moduleId)}>
                <Package className="size-4" />
                {item.release ? t('deploymentsPage.createRelease') : t('deploymentsPage.deployToEnvironment')}
              </Button>
              {item.release && (
                <>
                  <Button size="sm" variant="ghost" onClick={() => setLogRelease(item.release)}>
                    <Eye className="size-4" />
                    {t('deploymentsPage.viewLogs')}
                  </Button>
                  <Button disabled={item.release.status !== 'succeeded' || rollbackRelease.isPending} size="sm" variant="ghost" onClick={() => rollbackRelease.mutate(item.release.id)}>
                    <RotateCcw className="size-4" />
                    {t('deploymentsPage.rollback')}
                  </Button>
                </>
              )}
              <Button disabled={deleteTarget.isPending} size="sm" variant="ghost" onClick={() => setTargetToDelete(item.target)}>
                <Trash2 className="size-4" />
                {t('common.delete')}
              </Button>
            </div>
          ) },
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
            <Field hint={t('deploymentsPage.buildRunHint')} label={t('deploymentsPage.buildRun')} required>
              <Select {...form.register('buildRunId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {selectableBuildRuns.map(run => <option key={run.id} value={run.id}>{buildRunOptionLabel(run)}</option>)}
              </Select>
            </Field>
            <Field label={t('buildsPage.module')}>
              <Input readOnly value={modules.find(config => config.id === form.watch('moduleId'))?.name || form.watch('moduleId') || '-'} />
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
      <Dialog open={Boolean(logRelease)} onOpenChange={open => !open && setLogRelease(null)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>{t('deploymentsPage.releaseLogs')}</DialogTitle>
            <DialogDescription>{logRelease?.id}</DialogDescription>
          </DialogHeader>
          <pre className="max-h-[60vh] overflow-auto rounded-md border border-border bg-muted p-3 text-xs leading-relaxed text-foreground">
            {releaseLogs.data?.content || t('deploymentsPage.emptyLogs')}
          </pre>
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

function releaseEnvironmentLabel(environment: { name: string, stage?: string } | undefined, environmentID: string, t: ReturnType<typeof useTranslation>['t']) {
  if (!environment)
    return environmentID || '-'
  const stage = environment.stage ? t(`deploymentsPage.stageLabels.${environment.stage}`, { defaultValue: environment.stage }) : ''
  return stage ? `${environment.name} · ${stage}` : environment.name
}

function deploymentReleaseKey(environmentId: string, moduleId: string) {
  return `${environmentId}:${moduleId}`
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

function ApplicationGatewayPanel({ applicationId, applicationSlug, environments, projectId, ref, routes, servicePort }: {
  applicationId: string
  applicationSlug: string
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
  const saveRoute = useMutation({
    mutationFn: (values: RouteForm) => {
      const payload = { ...values, applicationId, applicationSlug, servicePort: values.servicePort || servicePort }
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
    form.reset(route ? { ...route, applicationSlug, stage: 'dev' } : { ...routeDefaults, applicationId, applicationSlug, servicePort })
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
              environmentIdField={form.register('environmentId')}
              environments={environments}
              hostField={form.register('host', { required: true })}
              pathField={form.register('path')}
              servicePortField={form.register('servicePort', { valueAsNumber: true })}
              stageField={form.register('stage')}
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

function ModuleBuildHookSelector({ hooks, loading, selectedIds, onSelectedIdsChange }: {
  hooks: ProjectHookConfig[]
  loading: boolean
  selectedIds: string[]
  onSelectedIdsChange: (ids: string[]) => void
}) {
  const { t } = useTranslation()
  const buildHooks = hooks.filter(hook => buildHookPhases.includes(hook.phase as Extract<ProjectHookConfig['phase'], 'preBuild' | 'postBuild'>))
  const hooksByID = new Map(buildHooks.map(hook => [hook.id, hook]))
  const selectedHooks = selectedIds.map(id => hooksByID.get(id)).filter((hook): hook is ProjectHookConfig => Boolean(hook))

  function toggleHook(hook: ProjectHookConfig, checked: boolean) {
    if (checked) {
      if (!selectedIds.includes(hook.id))
        onSelectedIdsChange([...selectedIds, hook.id])
      return
    }
    onSelectedIdsChange(selectedIds.filter(id => id !== hook.id))
  }

  function reorderPhase(phase: ProjectHookConfig['phase'], nextPhaseHooks: ProjectHookConfig[]) {
    const nextPhaseIds = nextPhaseHooks.map(hook => hook.id)
    const next = selectedIds.filter((id) => {
      const hook = hooksByID.get(id)
      return hook && hook.phase !== phase
    })
    const insertAt = phase === 'preBuild' ? 0 : next.filter(id => hooksByID.get(id)?.phase === 'preBuild').length
    next.splice(insertAt, 0, ...nextPhaseIds)
    onSelectedIdsChange(next)
  }

  if (loading && buildHooks.length === 0) {
    return <div className="rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">{t('common.loading')}</div>
  }

  if (buildHooks.length === 0) {
    return <EmptyState description={t('apps.moduleBuildHooksEmptyDescription')} title={t('apps.moduleBuildHooksEmptyTitle')} variant="plain" />
  }

  return (
    <div className="grid gap-4">
      {buildHookPhases.map((phase) => {
        const phaseHooks = buildHooks.filter(hook => hook.phase === phase)
        const selectedPhaseHooks = selectedHooks.filter(hook => hook.phase === phase)
        return (
          <div key={phase} className="grid gap-3 rounded-md border border-border p-3">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h4 className="text-sm font-medium">{t(`projectHooks.phases.${phase}`)}</h4>
                <p className="mt-1 text-xs text-muted-foreground">{t('apps.moduleBuildHooksPhaseHint')}</p>
              </div>
              <span className="text-xs text-muted-foreground">{t('apps.moduleBuildHooksSelectedCount', { count: selectedPhaseHooks.length })}</span>
            </div>
            <div className="grid gap-2 sm:grid-cols-2">
              {phaseHooks.map(hook => (
                <label key={hook.id} className="flex min-w-0 items-start gap-2 rounded-md border border-border bg-background px-3 py-2 text-sm">
                  <input
                    checked={selectedIds.includes(hook.id)}
                    className="mt-0.5 size-4 accent-primary"
                    type="checkbox"
                    onChange={event => toggleHook(hook, event.target.checked)}
                  />
                  <span className="min-w-0">
                    <span className="block truncate font-medium" title={hook.name}>{hook.name}</span>
                    <span className="mt-1 block truncate text-xs text-muted-foreground">
                      {`${t(`projectHooks.failurePolicies.${hook.failurePolicy}`)} · ${t(`projectHooks.shells.${hook.shell}`)} · ${hook.timeoutSeconds}s`}
                    </span>
                  </span>
                </label>
              ))}
            </div>
            {selectedPhaseHooks.length > 0 && (
              <SortableGrid
                className="grid-cols-1"
                getItemId={item => item.id}
                items={selectedPhaseHooks}
                renderItem={(hook, state) => (
                  <div className={`flex min-w-0 items-center gap-3 rounded-md border bg-muted/40 px-3 py-2 text-sm transition ${state.dragging ? 'opacity-60' : ''} ${state.insertBefore ? 'ring-2 ring-primary/40' : ''}`}>
                    <GripVertical className="size-4 shrink-0 cursor-grab text-muted-foreground" />
                    <span className="min-w-0 flex-1 truncate font-medium" title={hook.name}>{hook.name}</span>
                    <Button size="sm" type="button" variant="ghost" onClick={() => toggleHook(hook, false)}>
                      <X className="size-4" />
                      {t('common.remove')}
                    </Button>
                  </div>
                )}
                onOrderChange={items => reorderPhase(phase, items)}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}

function branchOptions(branches: Array<{ name: string }>, current?: string) {
  return withCurrentOption(branches.map(branch => branch.name), current)
}

function registryOptionLabel(registry: ArtifactRegistry) {
  return [registry.name, registry.provider].filter(Boolean).join(' · ')
}

function firstSelectableModule(configs: ApplicationModule[]) {
  return configs.find(config => config.enabled) ?? configs[0]
}

function moduleTargetImageRef(config?: ApplicationModule) {
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

function defaultTargetImageRef(registry: ArtifactRegistry | undefined, projectSlug: string, appSlug: string) {
  const imageName = [slugSegment(projectSlug), slugSegment(appSlug)].filter(Boolean).join('-')
  if (!imageName)
    return ''
  const namespace = registry?.namespace?.trim().replace(/^\/+|\/+$/g, '')
  return `${namespace ? `${namespace}/` : ''}${imageName}:latest`
}

function defaultModuleTargetImageRef(registry: ArtifactRegistry | undefined, appSlug: string, moduleSlug: string) {
  const imageName = [slugSegment(appSlug), slugSegment(moduleSlug)].filter(Boolean).join('-')
  if (!imageName)
    return ''
  const namespace = registry?.namespace?.trim().replace(/^\/+|\/+$/g, '')
  return `${namespace ? `${namespace}/` : ''}${imageName}:latest`
}

function dockerfileDirectory(path: string) {
  const normalized = path.trim().replace(/^\/+|\/+$/g, '')
  if (!normalized || normalized === 'Dockerfile')
    return '.'
  const index = normalized.lastIndexOf('/')
  return index > 0 ? normalized.slice(0, index) : '.'
}

function isDockerHubRegistry(registry: ArtifactRegistry) {
  if (registry.provider === 'dockerhub')
    return true
  const host = registryHost(registry.endpoint)
  return ['docker.io', 'registry-1.docker.io', 'index.docker.io'].includes(host)
}

function registryHost(endpoint: string) {
  try {
    return new URL(endpoint).host.toLowerCase()
  }
  catch {
    return endpoint.replace(/^https?:\/\//, '').replace(/\/.*$/, '').toLowerCase()
  }
}

function slugSegment(value: string) {
  return value.trim().replace(/^\/+|\/+$/g, '').toLowerCase()
}

function withCurrentOption(values: string[], current?: string) {
  const options = values.map(value => ({ value, label: value }))
  const normalized = current?.trim()
  if (normalized && !options.some(option => option.value === normalized))
    options.unshift({ value: normalized, label: normalized })
  return options
}
