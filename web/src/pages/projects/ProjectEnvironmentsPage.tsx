import type { Environment, RuntimeCluster } from '@/api/client'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, Trash2 } from 'lucide-react'
import { useImperativeHandle, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { FormField as Field } from '@/components/common/form-field'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { ENVIRONMENT_SLUG_MAX_LENGTH } from '@/lib/slug-limits'

export interface ProjectEnvironmentsPageHandle {
  openCreateDialog: () => void
}

type ResourceUnit = 'm' | 'core' | 'Mi' | 'Gi'
type EnvironmentPayload = Omit<Environment, 'id' | 'projectId' | 'createdBy' | 'createdAt'>
type EnvironmentForm = EnvironmentPayload & {
  cpuAmount: string
  cpuUnit: Extract<ResourceUnit, 'core' | 'm'>
  memoryAmount: string
  memoryUnit: Extract<ResourceUnit, 'Gi' | 'Mi'>
}

const environmentDefaults: EnvironmentForm = {
  clusterId: '',
  configRefs: '',
  cpuRequest: '100m',
  cpuAmount: '100',
  cpuUnit: 'm',
  envVars: '{}',
  memoryRequest: '128Mi',
  memoryAmount: '128',
  memoryUnit: 'Mi',
  name: '',
  namespace: '',
  replicas: 1,
  secretRefs: '',
  slug: '',
  stage: 'dev',
}

export function ProjectEnvironmentsPage({ projectId, ref }: { projectId: string, ref?: React.Ref<ProjectEnvironmentsPageHandle> }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingEnvironment, setEditingEnvironment] = useState<Environment | null>(null)
  const [environmentToDelete, setEnvironmentToDelete] = useState<Environment | null>(null)
  const environments = useQuery({ queryKey: ['environments', projectId], queryFn: () => api.listEnvironments(projectId), enabled: Boolean(projectId) })
  const clusters = useQuery({ queryKey: ['runtime-clusters', projectId], queryFn: () => api.listRuntimeClusters(projectId), enabled: Boolean(projectId) })
  const clusterMap = useMemo(() => Object.fromEntries((clusters.data ?? []).map(cluster => [cluster.id, cluster])), [clusters.data])
  const form = useForm<EnvironmentForm>({ defaultValues: environmentDefaults, mode: 'onChange' })

  useImperativeHandle(ref, () => ({
    openCreateDialog: () => openDialog(),
  }))

  const saveEnvironment = useMutation({
    mutationFn: (values: EnvironmentForm) => {
      const payload = environmentPayload(values)
      return editingEnvironment ? api.updateEnvironment(projectId, editingEnvironment.id, payload) : api.createEnvironment(projectId, payload)
    },
    onSuccess: () => {
      toast.success(t(editingEnvironment ? 'deploymentsPage.environmentUpdated' : 'deploymentsPage.environmentCreated'))
      setDialogOpen(false)
      setEditingEnvironment(null)
      form.reset(environmentDefaults)
      queryClient.invalidateQueries({ queryKey: ['environments', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteEnvironment = useMutation({
    mutationFn: (environmentId: string) => api.deleteEnvironment(projectId, environmentId),
    onSuccess: () => {
      toast.success(t('deploymentsPage.environmentDeleted'))
      setEnvironmentToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['environments', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  function openDialog(environment?: Environment) {
    setEditingEnvironment(environment ?? null)
    form.reset(environment ? environmentFormFromEnvironment(environment) : environmentDefaults)
    setDialogOpen(true)
  }

  return (
    <div className="grid gap-4">
      <DataList
        columns={[
          { key: 'name', header: t('common.name'), render: item => item.name },
          { key: 'stage', header: t('deploymentsPage.stage'), render: item => t(`deploymentsPage.stageLabels.${item.stage}`, { defaultValue: item.stage }) },
          { key: 'cluster', header: t('deploymentsPage.cluster'), render: item => clusterLabel(clusterMap[item.clusterId], item.clusterId, t) },
          { key: 'runtime', header: t('deploymentsPage.runtimeProfile'), render: item => `${item.replicas || 1} / ${item.cpuRequest || '-'} / ${item.memoryRequest || '-'}` },
          { key: 'actions', header: t('common.actions'), className: 'text-right whitespace-nowrap', render: item => (
            <div className="flex justify-end gap-2">
              <EditActionButton label={t('common.edit')} onClick={() => openDialog(item)} />
              <Button size="sm" variant="ghost" onClick={() => setEnvironmentToDelete(item)}>
                <Trash2 className="size-4" />
                {t('common.delete')}
              </Button>
            </div>
          ) },
        ]}
        emptyTitle={t('deploymentsPage.emptyEnvironments')}
        items={environments.data ?? []}
        rowKey={item => item.id}
      />

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingEnvironment ? t('deploymentsPage.editEnvironment') : t('deploymentsPage.createEnvironment')}</DialogTitle>
            <DialogDescription>{t('deploymentsPage.environmentDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={form.handleSubmit(values => saveEnvironment.mutate(values))}>
            <Field label={t('common.name')} required><Input {...form.register('name', { required: true })} /></Field>
            <Field hint={t('deploymentsPage.environmentSlugHint', { count: ENVIRONMENT_SLUG_MAX_LENGTH })} label={t('common.slug')} required>
              <Input {...form.register('slug', { maxLength: ENVIRONMENT_SLUG_MAX_LENGTH, required: true })} maxLength={ENVIRONMENT_SLUG_MAX_LENGTH} />
            </Field>
            <Field label={t('deploymentsPage.stage')}>
              <Select {...form.register('stage')}>
                <option value="dev">{t('deploymentsPage.stageDev')}</option>
                <option value="test">{t('deploymentsPage.stageTest')}</option>
                <option value="staging">{t('deploymentsPage.stageStaging')}</option>
                <option value="prod">{t('deploymentsPage.stageProd')}</option>
              </Select>
            </Field>
            <Field hint={t('deploymentsPage.clusterHint')} label={t('deploymentsPage.cluster')}>
              <Select {...form.register('clusterId')}>
                <option value="">{t('deploymentsPage.defaultCluster')}</option>
                {(clusters.data ?? []).map(cluster => <option key={cluster.id} value={cluster.id}>{clusterOptionLabel(cluster, t)}</option>)}
              </Select>
            </Field>
            <div className="grid gap-3 sm:grid-cols-3">
              <Field label={t('deploymentsPage.replicas')}><Input {...form.register('replicas', { valueAsNumber: true })} min={1} type="number" /></Field>
              <Field label={t('deploymentsPage.cpuRequest')}>
                <ResourceQuantityInput
                  amount={form.watch('cpuAmount')}
                  unitLabels={{
                    core: t('deploymentsPage.cpuUnits.core'),
                    m: t('deploymentsPage.cpuUnits.m'),
                  }}
                  unit={form.watch('cpuUnit')}
                  units={['m', 'core']}
                  onAmountChange={value => form.setValue('cpuAmount', value, { shouldDirty: true, shouldValidate: true })}
                  onUnitChange={value => form.setValue('cpuUnit', value as EnvironmentForm['cpuUnit'], { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
              <Field label={t('deploymentsPage.memoryRequest')}>
                <ResourceQuantityInput
                  amount={form.watch('memoryAmount')}
                  unitLabels={{ Gi: 'Gi', Mi: 'Mi' }}
                  unit={form.watch('memoryUnit')}
                  units={['Mi', 'Gi']}
                  onAmountChange={value => form.setValue('memoryAmount', value, { shouldDirty: true, shouldValidate: true })}
                  onUnitChange={value => form.setValue('memoryUnit', value as EnvironmentForm['memoryUnit'], { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            </div>
            <DialogFooter><Button disabled={!projectId || !form.formState.isValid || saveEnvironment.isPending} type="submit">{t('common.save')}</Button></DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('deploymentsPage.deleteEnvironmentDescription')}
        open={Boolean(environmentToDelete)}
        title={t('deploymentsPage.deleteEnvironmentTitle')}
        onConfirm={() => environmentToDelete && deleteEnvironment.mutate(environmentToDelete.id)}
        onOpenChange={open => !open && setEnvironmentToDelete(null)}
      />
    </div>
  )
}

function clusterLabel(cluster: RuntimeCluster | undefined, clusterID: string, t: (key: string) => string) {
  if (cluster)
    return cluster.name
  if (clusterID)
    return clusterID
  return t('deploymentsPage.defaultCluster')
}

function clusterOptionLabel(cluster: RuntimeCluster, t: (key: string, options?: Record<string, unknown>) => string) {
  if (cluster.isDefault)
    return t('deploymentsPage.clusterDefaultOption', { name: cluster.name })
  return cluster.name
}

function ResourceQuantityInput({ amount, onAmountChange, onUnitChange, unit, unitLabels, units }: {
  amount: string
  unit: ResourceUnit
  units: ResourceUnit[]
  unitLabels: Partial<Record<ResourceUnit, string>>
  onAmountChange: (value: string) => void
  onUnitChange: (value: ResourceUnit) => void
}) {
  return (
    <div className="flex h-9 overflow-hidden rounded-full border border-input bg-background shadow-xs transition focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50">
      <Input
        className="h-full min-w-0 flex-1 rounded-none border-0 bg-transparent px-4 shadow-none focus-visible:ring-0"
        inputMode="numeric"
        pattern="[0-9]*"
        value={normalizeResourceAmount(amount)}
        onChange={event => onAmountChange(normalizeResourceAmount(event.target.value))}
      />
      <div className="my-1 w-px bg-border" />
      <div className="relative h-full shrink-0">
        <select
          className="h-full appearance-none bg-transparent px-3 pr-8 text-sm text-muted-foreground outline-none"
          value={unit}
          onChange={event => onUnitChange(event.target.value as ResourceUnit)}
        >
          {units.map(item => <option key={item} value={item}>{unitLabels[item] ?? item}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
      </div>
    </div>
  )
}

function environmentFormFromEnvironment(environment: Environment): EnvironmentForm {
  const cpu = parseResourceQuantity(environment.cpuRequest, ['m', 'core'])
  const memory = parseResourceQuantity(environment.memoryRequest, ['Mi', 'Gi'])
  return {
    clusterId: environment.clusterId,
    configRefs: environment.configRefs,
    cpuRequest: environment.cpuRequest,
    cpuAmount: cpu.amount,
    cpuUnit: cpu.unit === 'core' ? 'core' : 'm',
    envVars: environment.envVars,
    memoryRequest: environment.memoryRequest,
    memoryAmount: memory.amount,
    memoryUnit: memory.unit === 'Gi' ? 'Gi' : 'Mi',
    name: environment.name,
    namespace: environment.namespace,
    replicas: environment.replicas,
    secretRefs: environment.secretRefs,
    slug: environment.slug,
    stage: environment.stage,
  }
}

function environmentPayload(values: EnvironmentForm): EnvironmentPayload {
  const {
    cpuAmount,
    cpuUnit,
    memoryAmount,
    memoryUnit,
    ...payload
  } = values
  return {
    ...payload,
    cpuRequest: formatResourceQuantity(cpuAmount, cpuUnit),
    memoryRequest: formatResourceQuantity(memoryAmount, memoryUnit),
  }
}

function parseResourceQuantity(value: string, units: ResourceUnit[]) {
  const normalized = String(value || '').trim()
  const fallbackUnit = units[0] ?? 'm'
  if (!normalized)
    return { amount: '', unit: fallbackUnit }
  if (normalized.endsWith('m') && units.includes('m'))
    return { amount: normalizeResourceAmount(normalized.slice(0, -1)) || '0', unit: 'm' as ResourceUnit }
  if (normalized.endsWith('Mi') && units.includes('Mi'))
    return { amount: normalizeResourceAmount(normalized.slice(0, -2)) || '0', unit: 'Mi' as ResourceUnit }
  if (normalized.endsWith('Gi') && units.includes('Gi'))
    return { amount: normalizeResourceAmount(normalized.slice(0, -2)) || '0', unit: 'Gi' as ResourceUnit }
  if (units.includes('core'))
    return { amount: normalizeResourceAmount(normalized), unit: 'core' as ResourceUnit }
  return { amount: normalizeResourceAmount(normalized), unit: fallbackUnit }
}

function formatResourceQuantity(amount: string, unit: ResourceUnit) {
  const normalized = normalizeResourceAmount(amount)
  if (!normalized)
    return ''
  if (unit === 'core')
    return normalized
  return `${normalized}${unit}`
}

function normalizeResourceAmount(value: string) {
  return value.replace(/\D/g, '')
}
