import type { ClusterResource, ClusterResourceEvent, CurrentUser, RuntimeCluster } from '@/api/client'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Plus, RefreshCcw, ScrollText, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { useSession } from '@/app/session-context'
import { CodeEditor } from '@/components/common/code-editor'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ContentTabs } from '@/components/common/content-tabs'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { EmptyState } from '@/components/common/empty-state'
import { FormField as Field } from '@/components/common/form-field'
import { StatusValueBadge } from '@/components/common/status-badge'
import { formatSmartDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { TabsContent } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

type ClusterForm = Omit<RuntimeCluster, 'id' | 'createdBy' | 'createdAt' | 'kubeconfigSet' | 'lastCheckedAt'> & { kubeconfig?: string }

const clusterDefaults: ClusterForm = {
  endpoint: '',
  isDefault: false,
  kubeconfig: '',
  name: '',
  ownerRef: '',
  scope: 'global',
  status: 'unknown',
  type: 'kubernetes',
}

export function ClustersPage() {
  const { t } = useTranslation()
  const { user } = useSession()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState('clusters')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingCluster, setEditingCluster] = useState<RuntimeCluster | null>(null)
  const [clusterToDelete, setClusterToDelete] = useState<RuntimeCluster | null>(null)
  const [resourceToDelete, setResourceToDelete] = useState<ClusterResource | null>(null)
  const [resourcesToDelete, setResourcesToDelete] = useState<ClusterResource[]>([])
  const [eventResource, setEventResource] = useState<ClusterResource | null>(null)
  const [selectedResourceClusterId, setSelectedResourceClusterId] = useState('')
  const [selectedResourceKeys, setSelectedResourceKeys] = useState<string[]>([])
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
  const clusters = useQuery({ queryKey: ['runtime-clusters'], queryFn: () => api.listRuntimeClusters() })
  const projectMap = useMemo(() => Object.fromEntries((projects.data ?? []).map(project => [project.id, project])), [projects.data])
  const manageableClusters = useMemo(() => (clusters.data ?? []).filter(cluster => canManageCluster(cluster, user?.id, user?.role)), [clusters.data, user?.id, user?.role])
  const effectiveResourceClusterId = manageableClusters.some(cluster => cluster.id === selectedResourceClusterId) ? selectedResourceClusterId : manageableClusters[0]?.id ?? ''
  const selectedResourceCluster = manageableClusters.find(cluster => cluster.id === effectiveResourceClusterId)
  const resourceKind = activeTab === 'clusters' ? 'namespaces' : activeTab
  const clusterResources = useQuery({
    queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind],
    queryFn: () => api.listRuntimeClusterResources(selectedResourceCluster?.id ?? '', { kind: resourceKind }),
    enabled: activeTab !== 'clusters' && Boolean(selectedResourceCluster?.id),
  })
  const activeResourceItems = useMemo(() => activeTab === 'clusters' ? [] : clusterResources.data ?? [], [activeTab, clusterResources.data])
  const activeResourceKeySet = useMemo(() => new Set(activeResourceItems.map(item => item.id)), [activeResourceItems])
  const visibleSelectedResourceKeys = useMemo(() => selectedResourceKeys.filter(key => activeResourceKeySet.has(key)), [activeResourceKeySet, selectedResourceKeys])
  const selectedDeletableResources = useMemo(() => {
    const selectedKeys = new Set(visibleSelectedResourceKeys)
    return activeResourceItems.filter(item => selectedKeys.has(item.id) && canDeleteClusterResource(user, item))
  }, [activeResourceItems, user, visibleSelectedResourceKeys])
  const resourceEvents = useQuery({
    queryKey: ['runtime-cluster-resource-events', selectedResourceCluster?.id, eventResource?.kind, eventResource?.namespace, eventResource?.name],
    queryFn: () => api.listRuntimeClusterResourceEvents(selectedResourceCluster?.id ?? '', {
      kind: eventResource?.kind ?? '',
      namespace: eventResource?.namespace,
      name: eventResource?.name ?? '',
    }),
    enabled: Boolean(selectedResourceCluster?.id && eventResource),
  })
  const form = useForm<ClusterForm>({ defaultValues: clusterDefaults, mode: 'onChange' })
  const scope = form.watch('scope')
  const canEditKubeconfig = !editingCluster || canInspectClusterKubeconfig(editingCluster, user?.id, user?.role)

  useEffect(() => {
    if (scope !== 'global')
      form.setValue('isDefault', false, { shouldDirty: true, shouldValidate: true })
    if (scope === 'user')
      form.setValue('ownerRef', '', { shouldDirty: true, shouldValidate: true })
  }, [form, scope])

  const saveCluster = useMutation({
    mutationFn: (values: ClusterForm) => editingCluster ? api.updateRuntimeCluster(editingCluster.id, values) : api.createRuntimeCluster(values),
    onSuccess: () => {
      toast.success(t(editingCluster ? 'deploymentsPage.clusterUpdated' : 'deploymentsPage.clusterCreated'))
      setDialogOpen(false)
      setEditingCluster(null)
      form.reset(clusterDefaults)
      queryClient.invalidateQueries({ queryKey: ['runtime-clusters'] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteCluster = useMutation({
    mutationFn: api.deleteRuntimeCluster,
    onSuccess: () => {
      toast.success(t('deploymentsPage.clusterDeleted'))
      setClusterToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['runtime-clusters'] })
    },
    onError: error => toast.error(error.message),
  })
  const testCluster = useMutation({
    mutationFn: api.testRuntimeCluster,
    onSuccess: () => {
      toast.success(t('deploymentsPage.clusterTested'))
    },
    onError: error => toast.error(error.message),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ['runtime-clusters'] }),
  })
  const deleteResource = useMutation({
    mutationFn: (resource: ClusterResource) => api.deleteRuntimeClusterResource(effectiveResourceClusterId, {
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
    }),
    onSuccess: () => {
      toast.success(t('clustersPage.resourceDeleted'))
      setResourceToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteResources = useMutation({
    mutationFn: async (resources: ClusterResource[]) => {
      for (const resource of resources) {
        await api.deleteRuntimeClusterResource(effectiveResourceClusterId, {
          kind: resource.kind,
          namespace: resource.namespace,
          name: resource.name,
        })
      }
    },
    onSuccess: (_, resources) => {
      toast.success(t('clustersPage.resourcesDeleted', { count: resources.length }))
      setResourcesToDelete([])
      setSelectedResourceKeys([])
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind] })
    },
    onError: error => toast.error(error.message),
  })

  function openDialog(cluster?: RuntimeCluster) {
    setEditingCluster(cluster ?? null)
    form.reset(cluster
      ? {
          endpoint: cluster.endpoint,
          isDefault: cluster.isDefault,
          kubeconfig: cluster.kubeconfig ?? '',
          name: cluster.name,
          ownerRef: cluster.ownerRef,
          scope: cluster.scope,
          status: cluster.status,
          type: cluster.type,
        }
      : clusterDefaults)
    setDialogOpen(true)
  }

  return (
    <div className="grid gap-4">
      <ContentTabs
        tabs={[
          { label: t('clustersPage.runtimeClustersTab'), value: 'clusters' },
          { label: t('clustersPage.namespacesTab'), value: 'namespaces' },
          { label: t('clustersPage.workloadsTab'), value: 'workloads' },
          { label: t('clustersPage.servicesTab'), value: 'services' },
          { label: t('clustersPage.configsTab'), value: 'configs' },
          { label: t('clustersPage.storageTab'), value: 'storage' },
        ]}
        tools={(
          <div className="flex flex-wrap items-center gap-2">
            {activeTab === 'clusters'
              ? (
                  <Button onClick={() => openDialog()}>
                    <Plus className="size-4" />
                    {t('deploymentsPage.createCluster')}
                  </Button>
                )
              : (
                  <>
                    <Select
                      aria-label={t('clustersPage.selectResourceCluster')}
                      className="h-9"
                      containerClassName="w-52 max-w-full"
                      disabled={manageableClusters.length === 0}
                      value={effectiveResourceClusterId}
                      onChange={(event) => {
                        setSelectedResourceClusterId(event.target.value)
                        setSelectedResourceKeys([])
                      }}
                    >
                      {manageableClusters.length > 0
                        ? manageableClusters.map(cluster => <option key={cluster.id} value={cluster.id}>{cluster.name}</option>)
                        : <option value="">{t('clustersPage.noManageableClusterTitle')}</option>}
                    </Select>
                    <Button disabled={!selectedResourceCluster || clusterResources.isFetching} variant="secondary" onClick={() => clusterResources.refetch()}>
                      <RefreshCcw className="size-4" />
                      {t('common.refresh')}
                    </Button>
                    {visibleSelectedResourceKeys.length > 0 && (
                      <span className="text-xs text-muted-foreground">
                        {t('clustersPage.selectedResources', { count: selectedDeletableResources.length })}
                      </span>
                    )}
                    <Button
                      disabled={selectedDeletableResources.length === 0 || deleteResources.isPending}
                      variant="destructive"
                      onClick={() => setResourcesToDelete(selectedDeletableResources)}
                    >
                      <Trash2 className="size-4" />
                      {t('clustersPage.deleteSelectedResources')}
                    </Button>
                  </>
                )}
          </div>
        )}
        value={activeTab}
        onValueChange={(value) => {
          setActiveTab(value)
          setSelectedResourceKeys([])
        }}
      >
        <TabsContent value="clusters">
          <DataList
            columns={[
              { key: 'name', header: t('common.name'), render: item => item.name },
              { key: 'type', header: t('common.type'), render: item => clusterTypeLabel(item.type, t) },
              { key: 'scope', header: t('common.scope'), render: item => scopeLabel(item, projectMap, t) },
              { key: 'default', header: t('clustersPage.defaultCluster'), render: item => item.isDefault ? t('common.yes') : t('common.no') },
              { key: 'status', header: t('common.status'), render: item => <StatusValueBadge value={item.status} /> },
              { key: 'actions', header: t('common.actions'), className: 'text-right whitespace-nowrap', render: item => (
                canManageCluster(item, user?.id, user?.role)
                  ? (
                      <div className="flex justify-end gap-2">
                        <Button size="sm" variant="ghost" onClick={() => testCluster.mutate(item.id)}>{t('common.test')}</Button>
                        <EditActionButton label={t('common.edit')} onClick={() => openDialog(item)} />
                        <Button size="sm" variant="ghost" onClick={() => setClusterToDelete(item)}>
                          <Trash2 className="size-4" />
                          {t('common.delete')}
                        </Button>
                      </div>
                    )
                  : <span className="text-xs text-muted-foreground">{t('common.viewOnly')}</span>
              ) },
            ]}
            emptyTitle={t('deploymentsPage.emptyClusters')}
            items={clusters.data ?? []}
            rowKey={item => item.id}
          />
        </TabsContent>
        {['namespaces', 'workloads', 'services', 'configs', 'storage'].map(tab => (
          <TabsContent key={tab} value={tab}>
            <ClusterResourcesPanel
              items={activeTab === tab ? clusterResources.data ?? [] : []}
              loading={activeTab === tab && clusterResources.isFetching}
              selectedCluster={selectedResourceCluster}
              selectedResourceKeys={activeTab === tab ? selectedResourceKeys : []}
              tab={tab}
              user={user}
              onDeleteResource={setResourceToDelete}
              onOpenEvents={setEventResource}
              onSelectionChange={setSelectedResourceKeys}
            />
          </TabsContent>
        ))}
      </ContentTabs>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="flex max-h-[min(88vh,52rem)] w-[min(92vw,48rem)] max-w-[92vw] min-w-0 flex-col gap-0 overflow-hidden p-0">
          <DialogHeader className="shrink-0 border-b border-border p-5 pb-4">
            <DialogTitle>{editingCluster ? t('deploymentsPage.editCluster') : t('deploymentsPage.createCluster')}</DialogTitle>
            <DialogDescription>{t('clustersPage.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="flex min-h-0 min-w-0 flex-1 flex-col" onSubmit={form.handleSubmit(values => saveCluster.mutate(values))}>
            <div className="min-h-0 min-w-0 max-w-full flex-1 overflow-y-auto overflow-x-hidden px-5 py-4">
              <div className="grid min-w-0 max-w-full gap-3 overflow-x-hidden">
                <Field label={t('common.name')} required><Input {...form.register('name', { required: true })} /></Field>
                <Field label={t('common.scope')}>
                  <Select {...form.register('scope')}>
                    <option value="global">{t('codeRepositoriesView.scopeGlobal')}</option>
                    <option value="project">{t('codeRepositoriesView.scopeProject')}</option>
                    <option value="user">{t('codeRepositoriesView.scopeUser')}</option>
                  </Select>
                </Field>
                {scope === 'project' && (
                  <Field label={t('projectSpaces.title')} required>
                    <Select {...form.register('ownerRef', { required: true })}>
                      <option value="">{t('common.select')}</option>
                      {(projects.data ?? []).map(project => <option key={project.id} value={project.id}>{project.name}</option>)}
                    </Select>
                  </Field>
                )}
                <Field label={t('common.type')}>
                  <Select {...form.register('type')}>
                    <option value="kubernetes">{t('deploymentsPage.typeKubernetes')}</option>
                  </Select>
                </Field>
                <Field hint={canEditKubeconfig ? t('clustersPage.kubeconfigHint') : t('clustersPage.kubeconfigRestrictedHint')} label={t('deploymentsPage.kubeconfig')} required={!editingCluster}>
                  <Controller
                    control={form.control}
                    name="kubeconfig"
                    rules={{ required: !editingCluster }}
                    render={({ field }) => (
                      <div className="min-w-0 max-w-full overflow-x-hidden">
                        <CodeEditor
                          ariaInvalid={Boolean(form.formState.errors.kubeconfig)}
                          className="w-full"
                          height="22rem"
                          language="yaml"
                          placeholder={t('clustersPage.kubeconfigPlaceholder')}
                          readOnly={!canEditKubeconfig}
                          value={field.value ?? ''}
                          onChange={field.onChange}
                        />
                      </div>
                    )}
                  />
                </Field>
                {scope === 'global' && (
                  <label className="flex items-center gap-2 text-sm text-foreground">
                    <input className="size-4 accent-primary" type="checkbox" {...form.register('isDefault')} />
                    <span>{t('clustersPage.defaultCluster')}</span>
                  </label>
                )}
              </div>
            </div>
            <DialogFooter className="shrink-0 border-t border-border p-5 pt-4">
              <Button disabled={!form.formState.isValid || saveCluster.isPending} type="submit">{t('common.save')}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog cancelText={t('common.cancel')} confirmText={t('common.delete')} description={t('deploymentsPage.deleteClusterDescription')} open={Boolean(clusterToDelete)} title={t('deploymentsPage.deleteClusterTitle')} onConfirm={() => clusterToDelete && deleteCluster.mutate(clusterToDelete.id)} onOpenChange={open => !open && setClusterToDelete(null)} />
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('clustersPage.deleteResourceDescription', { kind: resourceToDelete?.kind ?? '', namespace: resourceToDelete?.namespace || '-', name: resourceToDelete?.name ?? '' })}
        open={Boolean(resourceToDelete)}
        pending={deleteResource.isPending}
        title={t('clustersPage.deleteResourceTitle')}
        onConfirm={() => resourceToDelete && deleteResource.mutate(resourceToDelete)}
        onOpenChange={open => !open && setResourceToDelete(null)}
      />
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('clustersPage.deleteResourcesDescription', { count: resourcesToDelete.length })}
        open={resourcesToDelete.length > 0}
        pending={deleteResources.isPending}
        title={t('clustersPage.deleteResourcesTitle')}
        onConfirm={() => {
          if (resourcesToDelete.length > 0) {
            deleteResources.mutate(resourcesToDelete)
          }
        }}
        onOpenChange={open => !open && setResourcesToDelete([])}
      />
      <Dialog open={Boolean(eventResource)} onOpenChange={open => !open && setEventResource(null)}>
        <DialogContent className="flex max-h-[min(88vh,42rem)] w-[min(92vw,56rem)] max-w-[92vw] min-w-0 flex-col overflow-hidden p-0">
          <DialogHeader className="shrink-0 border-b border-border p-5 pb-4">
            <DialogTitle>{t('clustersPage.resourceEventsTitle')}</DialogTitle>
            <DialogDescription>
              {eventResource ? t('clustersPage.resourceEventsDescription', { kind: eventResource.kind, namespace: eventResource.namespace || '-', name: eventResource.name }) : ''}
            </DialogDescription>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto p-5">
            <ClusterResourceEventsList events={resourceEvents.data ?? []} loading={resourceEvents.isFetching} />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function ClusterResourcesPanel({ items, loading, selectedCluster, selectedResourceKeys, tab, user, onDeleteResource, onOpenEvents, onSelectionChange }: {
  items: ClusterResource[]
  loading: boolean
  selectedCluster?: RuntimeCluster
  selectedResourceKeys: string[]
  tab: string
  user?: CurrentUser
  onDeleteResource: (resource: ClusterResource) => void
  onOpenEvents: (resource: ClusterResource) => void
  onSelectionChange: (keys: string[]) => void
}) {
  const { t } = useTranslation()
  const itemKeys = new Set(items.map(item => item.id))
  const visibleSelectedResourceKeys = selectedResourceKeys.filter(key => itemKeys.has(key))
  const selectedResources = items.filter(item => visibleSelectedResourceKeys.includes(item.id) && canDeleteClusterResource(user, item))
  if (!selectedCluster) {
    return (
      <EmptyState
        title={t('clustersPage.noManageableClusterTitle')}
        description={t('clustersPage.noManageableClusterDescription')}
      />
    )
  }
  return (
    <DataList
      columns={[
        { key: 'kind', header: t('clustersPage.resourceKind'), className: 'w-32 whitespace-nowrap', render: item => item.kind },
        { key: 'name', header: t('common.name'), className: 'min-w-56 whitespace-nowrap', render: item => <TruncatedResourceText className="max-w-72 font-mono text-sm" value={clusterResourceName(item, tab === 'namespaces')} /> },
        ...(tab === 'namespaces'
          ? []
          : [{ key: 'namespace', header: t('deploymentsPage.namespace'), className: 'w-44 whitespace-nowrap', render: (item: ClusterResource) => <TruncatedResourceText className="max-w-44 font-mono text-sm" value={item.namespace || '-'} /> }]),
        { key: 'status', header: t('common.status'), className: 'w-28 whitespace-nowrap', render: item => <StatusValueBadge value={normalizeClusterResourceStatus(item.status)} /> },
        { key: 'owner', header: t('clustersPage.resourceOwner'), className: 'min-w-56', render: item => <TruncatedResourceText className="max-w-72 text-sm" value={clusterResourceOwner(item)} /> },
        { key: 'summary', header: t('clustersPage.resourceSummary'), className: 'min-w-64', render: item => <TruncatedResourceText className="max-w-80 text-sm text-muted-foreground" value={item.summary || '-'} /> },
        {
          key: 'actions',
          header: t('common.actions'),
          className: 'w-40 whitespace-nowrap text-right',
          render: item => (
            <div className="flex justify-end gap-2">
              <Button size="sm" variant="ghost" onClick={() => onOpenEvents(item)}>
                <ScrollText className="size-4" />
                {t('clustersPage.viewEvents')}
              </Button>
              {canDeleteClusterResource(user, item) && (
                <Button size="sm" variant="ghost" onClick={() => onDeleteResource(item)}>
                  <Trash2 className="size-4" />
                  {t('common.delete')}
                </Button>
              )}
            </div>
          ),
        },
      ]}
      emptyDescription={loading ? t('common.loading') : t('clustersPage.resourceEmptyDescription')}
      emptyTitle={loading ? t('common.loading') : t(`clustersPage.${tab}EmptyTitle`)}
      items={items}
      rowKey={item => item.id}
      selection={{
        isRowSelectable: item => canDeleteClusterResource(user, item),
        selectAllLabel: t('clustersPage.selectAllResources'),
        selectedKeys: visibleSelectedResourceKeys,
        selectedLabel: t('clustersPage.selectedResources', { count: selectedResources.length }),
        selectRowLabel: item => t('clustersPage.selectResource', { name: clusterResourceDisplayName(item) }),
        onSelectionChange,
      }}
    />
  )
}

function TruncatedResourceText({ className = 'max-w-56', value }: { className?: string, value: string }) {
  const { t } = useTranslation()
  const content = value || '-'
  const copyValue = () => {
    if (!content || content === '-')
      return
    navigator.clipboard.writeText(content)
      .then(() => toast.success(t('common.copied')))
      .catch(error => toast.error(error.message))
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`block truncate ${className}`} title={content}>
          {content}
        </span>
      </TooltipTrigger>
      <TooltipContent className="flex max-w-96 items-start gap-2 break-all leading-5" side="top">
        <button
          aria-label={t('common.copy')}
          className="mt-0.5 inline-flex size-5 shrink-0 items-center justify-center rounded-sm text-background/80 transition hover:bg-background/15 hover:text-background"
          type="button"
          onClick={copyValue}
        >
          <Copy className="size-3.5" />
        </button>
        <span>{content}</span>
      </TooltipContent>
    </Tooltip>
  )
}

function clusterResourceDisplayName(item: ClusterResource) {
  if (item.namespace?.trim())
    return `${item.namespace}/${item.name}`
  return item.name
}

function clusterResourceName(item: ClusterResource, includeNamespace: boolean) {
  if (includeNamespace)
    return clusterResourceDisplayName(item)
  return item.name
}

function ClusterResourceEventsList({ events, loading }: { events: ClusterResourceEvent[], loading: boolean }) {
  const { t } = useTranslation()
  if (loading) {
    return (
      <EmptyState
        title={t('common.loading')}
        description={t('clustersPage.resourceEventsLoading')}
      />
    )
  }
  if (events.length === 0) {
    return (
      <EmptyState
        title={t('clustersPage.resourceEventsEmptyTitle')}
        description={t('clustersPage.resourceEventsEmptyDescription')}
      />
    )
  }
  return (
    <div className="grid gap-3">
      {events.map(event => (
        <div key={event.id} className="rounded-md border border-border bg-surface-subtle p-3">
          <div className="flex flex-wrap items-center gap-2">
            <StatusValueBadge value={event.type || 'normal'} />
            <span className="font-medium text-foreground">{event.reason || t('common.none')}</span>
            <span className="text-xs text-muted-foreground">{formatSmartDateTime(event.lastSeen, t)}</span>
            {event.count > 1 && <span className="text-xs text-muted-foreground">{t('clustersPage.resourceEventCount', { count: event.count })}</span>}
          </div>
          <p className="mt-2 break-words text-sm text-foreground">{event.message || '-'}</p>
          <div className="mt-2 text-xs text-muted-foreground">
            {event.source || t('common.none')}
          </div>
        </div>
      ))}
    </div>
  )
}

function clusterResourceOwner(item: ClusterResource) {
  const project = item.projectName?.trim() || item.projectId?.trim()
  const application = item.applicationName?.trim() || item.applicationId?.trim()
  return [project, application].filter(Boolean).join(' / ') || '-'
}

function normalizeClusterResourceStatus(status: string) {
  const value = status.toLowerCase()
  if (value === 'running' || value === 'ready' || value === 'active' || value === 'bound')
    return 'ready'
  if (value === 'failed' || value === 'pending')
    return value
  return status || 'unknown'
}

function canDeleteClusterResource(user: CurrentUser | undefined, item: ClusterResource) {
  return user?.role === 'platform_admin' || Boolean(item.projectId?.trim())
}

function canManageCluster(cluster: RuntimeCluster, userID?: string, role?: string) {
  if (role === 'platform_admin')
    return true
  if (cluster.scope === 'user')
    return cluster.ownerRef === userID
  if (cluster.scope === 'project')
    return true
  return false
}

function canInspectClusterKubeconfig(cluster: RuntimeCluster, userID?: string, role?: string) {
  return role === 'platform_admin' || cluster.createdBy === userID
}

function clusterTypeLabel(type: RuntimeCluster['type'], t: (key: string, options?: Record<string, unknown>) => string) {
  if (type === 'k3s')
    return t('deploymentsPage.typeKubernetes')
  return t(`deploymentsPage.typeLabels.${type}`, { defaultValue: type })
}

function scopeLabel(cluster: RuntimeCluster, projectMap: Record<string, { name: string }>, t: (key: string, options?: Record<string, unknown>) => string) {
  if (cluster.scope === 'project')
    return projectMap[cluster.ownerRef]?.name ?? cluster.ownerRef
  if (cluster.scope === 'user')
    return t('codeRepositoriesView.scopeUser')
  return t('codeRepositoriesView.scopeGlobal')
}
