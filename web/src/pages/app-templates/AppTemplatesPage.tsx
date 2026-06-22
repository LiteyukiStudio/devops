import type { AppTemplate, AppTemplateInstallPayload, Project, RuntimeCluster } from '@/api/client'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Box, Database, PackageOpen, Rocket, Search } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ProjectSpaceSelect } from '@/components/common/project-space-select'
import { StatusBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { cn } from '@/lib/utils'

const FALLBACK_ICON = '/app-templates/icons/fallback.svg'

export function AppTemplatesPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<AppTemplate | null>(null)
  const [projectId, setProjectId] = useState('')
  const [form, setForm] = useState<AppTemplateInstallPayload>(emptyInstallPayload())

  const templates = useQuery({ queryKey: ['app-templates'], queryFn: api.listAppTemplates })
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
  const clusters = useQuery({
    queryKey: ['runtime-clusters', projectId],
    queryFn: () => api.listRuntimeClusters(projectId),
    enabled: Boolean(projectId),
  })
  const projectItems = projects.data ?? []
  const clusterItems = clusters.data ?? []

  useEffect(() => {
    if (!projectId && projectItems.length > 0)
      setProjectId(projectItems[0].id)
  }, [projectId, projectItems])

  const filteredTemplates = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    const items = templates.data ?? []
    if (!keyword)
      return items
    return items.filter(template => [template.name, template.slug, template.image, template.category]
      .some(value => value.toLowerCase().includes(keyword)))
  }, [search, templates.data])

  const installTemplate = useMutation({
    mutationFn: (payload: AppTemplateInstallPayload & { templateId: string, projectId: string }) =>
      api.installAppTemplate(payload.projectId, payload.templateId, payload),
    onSuccess: async (result) => {
      toast.success(t('appTemplatesPage.installStarted'))
      setSelectedTemplate(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['projects'] }),
        queryClient.invalidateQueries({ queryKey: ['applications', result.application.projectId] }),
      ])
      navigate(`/projects/${result.application.projectId}/apps/${result.application.id}#tab=deployments`)
    },
    onError: error => toast.error(error.message),
  })

  function openInstallDialog(template: AppTemplate) {
    setSelectedTemplate(template)
    setForm(payloadFromTemplate(template))
  }

  function updateForm<K extends keyof AppTemplateInstallPayload>(key: K, value: AppTemplateInstallPayload[K]) {
    setForm(current => ({ ...current, [key]: value }))
  }

  function updateTemplateValue(key: string, value: string) {
    setForm(current => ({ ...current, values: { ...current.values, [key]: value } }))
  }

  function submitInstall() {
    if (!selectedTemplate || !projectId)
      return
    installTemplate.mutate({ ...form, projectId, templateId: selectedTemplate.id })
  }

  return (
    <div className="grid gap-5">
      <Card className="grid gap-4 p-4 md:grid-cols-[minmax(0,1fr)_18rem] md:items-center">
        <div className="min-w-0">
          <h2 className="text-base font-semibold">{t('appTemplatesPage.heroTitle')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('appTemplatesPage.heroDescription')}</p>
        </div>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="h-11 rounded-full pl-10"
            placeholder={t('appTemplatesPage.searchPlaceholder')}
            value={search}
            onChange={event => setSearch(event.target.value)}
          />
        </div>
      </Card>

      {templates.isError && <ErrorState title={templates.error.message} />}
      {templates.isLoading && <EmptyState title={t('appTemplatesPage.loading')} variant="plain" />}
      {templates.isSuccess && filteredTemplates.length === 0 && (
        <EmptyState description={t('appTemplatesPage.emptyDescription')} title={t('appTemplatesPage.emptyTitle')} />
      )}
      {filteredTemplates.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {filteredTemplates.map(template => (
            <TemplateCard key={template.id} template={template} onInstall={() => openInstallDialog(template)} />
          ))}
        </div>
      )}

      <InstallTemplateDialog
        clusterItems={clusterItems}
        clustersLoading={clusters.isLoading}
        form={form}
        installing={installTemplate.isPending}
        projectId={projectId}
        projects={projectItems}
        template={selectedTemplate}
        onClose={() => setSelectedTemplate(null)}
        onProjectChange={setProjectId}
        onSubmit={submitInstall}
        onTemplateValueChange={updateTemplateValue}
        onUpdate={updateForm}
      />
    </div>
  )
}

function TemplateCard({ template, onInstall }: { template: AppTemplate, onInstall: () => void }) {
  const { t } = useTranslation()
  const CategoryIcon = template.category === 'database' ? Database : Box
  return (
    <Card className="flex min-h-56 flex-col gap-4 p-5">
      <div className="flex items-start gap-4">
        <div className="flex size-14 shrink-0 items-center justify-center rounded-xl border border-border bg-surface">
          <img
            alt=""
            className="size-9 object-contain"
            src={template.icon || FALLBACK_ICON}
            onError={(event) => {
              event.currentTarget.src = FALLBACK_ICON
            }}
          />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-lg font-semibold">{template.name}</h2>
            <StatusBadge tone="neutral">{t(`appTemplatesPage.categories.${template.category}`, { defaultValue: template.category })}</StatusBadge>
          </div>
          <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
            {t(`appTemplatesPage.templates.${template.id}.description`, { defaultValue: template.description || t('common.noDescription') })}
          </p>
        </div>
      </div>
      <div className="grid gap-2 text-sm text-muted-foreground">
        <TemplateFact label={t('appTemplatesPage.image')} value={template.image} />
        <TemplateFact label={t('appTemplatesPage.port')} value={String(template.servicePort)} />
        <TemplateFact label={t('appTemplatesPage.resources')} value={`${template.defaultCPU} / ${template.defaultMemory}`} />
      </div>
      <div className="mt-auto flex items-center justify-between gap-3">
        <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
          <CategoryIcon className="size-4" />
          {template.version}
        </span>
        <Button className="rounded-full" type="button" onClick={onInstall}>
          <Rocket className="size-4" />
          {t('appTemplatesPage.install')}
        </Button>
      </div>
    </Card>
  )
}

function TemplateFact({ label, value }: { label: string, value: string }) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-3">
      <span className="shrink-0">{label}</span>
      <span className="min-w-0 truncate font-mono text-foreground">{value}</span>
    </div>
  )
}

function InstallTemplateDialog({
  clusterItems,
  clustersLoading,
  form,
  installing,
  projectId,
  projects,
  template,
  onClose,
  onProjectChange,
  onSubmit,
  onTemplateValueChange,
  onUpdate,
}: {
  clusterItems: RuntimeCluster[]
  clustersLoading: boolean
  form: AppTemplateInstallPayload
  installing: boolean
  projectId: string
  projects: Project[]
  template: AppTemplate | null
  onClose: () => void
  onProjectChange: (value: string) => void
  onSubmit: () => void
  onTemplateValueChange: (key: string, value: string) => void
  onUpdate: <K extends keyof AppTemplateInstallPayload>(key: K, value: AppTemplateInstallPayload[K]) => void
}) {
  const { t } = useTranslation()
  const canSubmit = Boolean(template && projectId && form.applicationName.trim() && form.applicationSlug.trim() && !installing)
  return (
    <Dialog open={Boolean(template)} onOpenChange={open => !open && onClose()}>
      <DialogContent className="max-h-[min(92vh,54rem)] max-w-4xl overflow-hidden p-0">
        <DialogHeader className="border-b border-border px-6 py-5">
          <DialogTitle>{t('appTemplatesPage.installDialogTitle', { name: template?.name ?? '' })}</DialogTitle>
          <DialogDescription>{t('appTemplatesPage.installDialogDescription')}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 overflow-y-auto px-6 py-5">
          <div className="grid gap-5 md:grid-cols-2">
            <Field label={t('projectSpaces.title')}>
              <ProjectSpaceSelect
                disabled={projects.length === 0 || installing}
                projects={projects}
                value={projectId}
                onChange={onProjectChange}
              />
            </Field>
            <Field label={t('appTemplatesPage.runtimeCluster')}>
              <Select
                disabled={clustersLoading || installing}
                value={form.clusterId}
                onChange={event => onUpdate('clusterId', event.target.value)}
              >
                <option value="">{t('appTemplatesPage.defaultCluster')}</option>
                {clusterItems.map(cluster => (
                  <option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                    {cluster.isDefault ? ` (${t('common.default')})` : ''}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label={t('appTemplatesPage.applicationName')} required>
              <Input value={form.applicationName} onChange={event => onUpdate('applicationName', event.target.value)} />
            </Field>
            <Field label={t('appTemplatesPage.applicationSlug')} required>
              <Input value={form.applicationSlug} onChange={event => onUpdate('applicationSlug', normalizeSlugInput(event.target.value))} />
            </Field>
            <Field label={t('appTemplatesPage.deploymentName')}>
              <Input value={form.deploymentName} onChange={event => onUpdate('deploymentName', event.target.value)} />
            </Field>
            <Field label={t('appTemplatesPage.stage')}>
              <Select value={form.stage} onChange={event => onUpdate('stage', event.target.value)}>
                {['prod', 'staging', 'test', 'dev'].map(stage => (
                  <option key={stage} value={stage}>{t(`appTemplatesPage.stageOptions.${stage}`)}</option>
                ))}
              </Select>
            </Field>
            <Field label={t('appTemplatesPage.replicas')}>
              <Input min={1} type="number" value={form.replicas} onChange={event => onUpdate('replicas', Number(event.target.value || 1))} />
            </Field>
            <Field label={t('appTemplatesPage.cpu')}>
              <Input value={form.cpuRequest} onChange={event => onUpdate('cpuRequest', event.target.value)} />
            </Field>
            <Field label={t('appTemplatesPage.memory')}>
              <Input value={form.memoryRequest} onChange={event => onUpdate('memoryRequest', event.target.value)} />
            </Field>
            <Field label={t('appTemplatesPage.dataCapacity')}>
              <Input disabled={!template?.dataRetentionEnabled} value={form.dataCapacity} onChange={event => onUpdate('dataCapacity', event.target.value)} />
            </Field>
          </div>

          {template && template.values.length > 0 && (
            <div className="mt-6 grid gap-4 border-t border-border pt-5">
              <div>
                <h3 className="font-semibold">{t('appTemplatesPage.templateParameters')}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{t('appTemplatesPage.templateParametersDescription')}</p>
              </div>
              <div className="grid gap-5 md:grid-cols-2">
                {template.values.map(value => (
                  <Field
                    key={value.key}
                    label={t(`appTemplatesPage.valueLabels.${value.key}`, { defaultValue: value.label || value.key })}
                    required={value.required && !value.autoGenerate}
                  >
                    <Input
                      placeholder={value.autoGenerate ? t('appTemplatesPage.autoGeneratePlaceholder') : value.default}
                      type={value.secret ? 'password' : 'text'}
                      value={form.values[value.key] ?? ''}
                      onChange={event => onTemplateValueChange(value.key, event.target.value)}
                    />
                  </Field>
                ))}
              </div>
            </div>
          )}

          <label className="mt-6 flex items-start gap-3 rounded-2xl border border-border p-4 text-sm">
            <input
              checked={form.installNow}
              className="mt-1 size-4 accent-primary"
              disabled={installing}
              type="checkbox"
              onChange={event => onUpdate('installNow', event.target.checked)}
            />
            <span>
              <span className="block font-medium">{t('appTemplatesPage.installNow')}</span>
              <span className="mt-1 block text-muted-foreground">{t('appTemplatesPage.installNowDescription')}</span>
            </span>
          </label>
        </div>
        <DialogFooter className="border-t border-border px-6 py-4">
          <Button disabled={installing} type="button" variant="outline" onClick={onClose}>{t('common.cancel')}</Button>
          <Button disabled={!canSubmit} type="button" onClick={onSubmit}>
            <PackageOpen className={cn('size-4', installing && 'animate-pulse')} />
            {installing ? t('appTemplatesPage.installing') : t('appTemplatesPage.install')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Field({ children, label, required }: { children: React.ReactNode, label: string, required?: boolean }) {
  return (
    <div className="grid gap-2">
      <Label>
        {label}
        {required && <span className="ml-1 text-primary">*</span>}
      </Label>
      {children}
    </div>
  )
}

function emptyInstallPayload(): AppTemplateInstallPayload {
  return {
    applicationName: '',
    applicationSlug: '',
    deploymentName: 'default',
    stage: 'prod',
    clusterId: '',
    namespace: '',
    replicas: 1,
    cpuRequest: '1',
    memoryRequest: '1Gi',
    dataCapacity: '',
    installNow: true,
    values: {},
  }
}

function payloadFromTemplate(template: AppTemplate): AppTemplateInstallPayload {
  const suffix = Math.random().toString(36).slice(2, 8)
  return {
    ...emptyInstallPayload(),
    applicationName: template.name,
    applicationSlug: normalizeSlugInput(`${template.slug}-${suffix}`).slice(0, 20),
    replicas: template.defaultReplicas || 1,
    cpuRequest: template.defaultCPU || '1',
    memoryRequest: template.defaultMemory || '1Gi',
    dataCapacity: template.dataCapacity,
    values: Object.fromEntries(template.values.filter(value => !value.autoGenerate).map(value => [value.key, value.default])),
  }
}

function normalizeSlugInput(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-+/, '')
}
