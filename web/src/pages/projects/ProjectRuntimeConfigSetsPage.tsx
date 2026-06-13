import type { Ref } from 'react'
import type { ProjectRuntimeConfigSet, ProjectRuntimeConfigSetPayload } from '@/api/client'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FileCode2, Trash2 } from 'lucide-react'
import { useImperativeHandle, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api/client'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { RuntimeConfigFilesEditor } from '@/components/common/runtime-config-files-editor'
import { StatusValueBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { runtimeConfigFileCount } from '@/lib/runtime-config-files'

export interface ProjectRuntimeConfigSetsPageHandle {
  openCreateDialog: () => void
}

const runtimeConfigDefaults: ProjectRuntimeConfigSetPayload = {
  configFiles: '',
  enabled: true,
  envVars: '',
  name: '',
  secretFiles: '',
  secretRefs: '',
}

export function ProjectRuntimeConfigSetsPage({ projectId, ref }: { projectId: string, ref?: Ref<ProjectRuntimeConfigSetsPageHandle> }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingSet, setEditingSet] = useState<ProjectRuntimeConfigSet | null>(null)
  const [configFilesValid, setConfigFilesValid] = useState(true)
  const [secretFilesValid, setSecretFilesValid] = useState(true)
  const [setToDelete, setSetToDelete] = useState<ProjectRuntimeConfigSet | null>(null)
  const form = useForm<ProjectRuntimeConfigSetPayload>({ defaultValues: runtimeConfigDefaults, mode: 'onChange' })
  const configSets = useQuery({ queryKey: ['runtime-config-sets', projectId], queryFn: () => api.listProjectRuntimeConfigSets(projectId), enabled: Boolean(projectId) })

  const saveConfigSet = useMutation({
    mutationFn: (values: ProjectRuntimeConfigSetPayload) => editingSet
      ? api.updateProjectRuntimeConfigSet(projectId, editingSet.id, normalizeRuntimeConfigPayload(values))
      : api.createProjectRuntimeConfigSet(projectId, normalizeRuntimeConfigPayload(values)),
    onSuccess: (set) => {
      toast.success(t(editingSet ? 'runtimeConfigSets.updated' : 'runtimeConfigSets.created'))
      if (editingSet && (set.affectedDeploymentTargetCount ?? 0) > 0)
        toast.warning(t('runtimeConfigSets.updatedNeedsRestart', { count: set.affectedDeploymentTargetCount }))
      setDialogOpen(false)
      setEditingSet(null)
      form.reset(runtimeConfigDefaults)
      queryClient.invalidateQueries({ queryKey: ['runtime-config-sets', projectId] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteConfigSet = useMutation({
    mutationFn: (set: ProjectRuntimeConfigSet) => api.deleteProjectRuntimeConfigSet(projectId, set.id),
    onSuccess: () => {
      toast.success(t('runtimeConfigSets.deleted'))
      setSetToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['runtime-config-sets', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  function openDialog(set?: ProjectRuntimeConfigSet) {
    setEditingSet(set ?? null)
    setConfigFilesValid(true)
    setSecretFilesValid(true)
    form.reset(set
      ? {
          configFiles: set.configFiles,
          enabled: set.enabled,
          envVars: set.envVars,
          name: set.name,
          secretFiles: '',
          secretRefs: '',
        }
      : runtimeConfigDefaults)
    setDialogOpen(true)
  }

  useImperativeHandle(ref, () => ({
    openCreateDialog: () => {
      setEditingSet(null)
      setConfigFilesValid(true)
      setSecretFilesValid(true)
      form.reset(runtimeConfigDefaults)
      setDialogOpen(true)
    },
  }), [form])

  if (configSets.isError) {
    return (
      <ErrorState
        description={t('runtimeConfigSets.loadFailedDescription')}
        title={t('runtimeConfigSets.loadFailedTitle')}
      />
    )
  }

  return (
    <Card className="min-w-0 overflow-hidden p-0">
      <div className="border-b border-border px-4 py-4">
        <div className="min-w-0">
          <h2 className="text-base font-semibold">{t('runtimeConfigSets.title')}</h2>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">{t('runtimeConfigSets.description')}</p>
        </div>
      </div>
      <DataList
        columns={[
          { key: 'name', header: t('common.name'), className: 'min-w-48 px-4 py-3 align-middle', render: item => <span className="block truncate whitespace-nowrap" title={item.name}>{item.name}</span> },
          { key: 'configFiles', header: t('runtimeConfigSets.configFiles'), className: 'w-32 whitespace-nowrap px-4 py-3 align-middle', render: item => t('runtimeConfigSets.configFileState', { count: runtimeConfigFileCount(item.configFiles) }) },
          { key: 'secretFiles', header: t('runtimeConfigSets.secretFiles'), className: 'w-32 whitespace-nowrap px-4 py-3 align-middle', render: item => item.secretFilesSet ? t('runtimeConfigSets.configured') : t('runtimeConfigSets.notConfigured') },
          { key: 'enabled', header: t('common.status'), className: 'w-28 whitespace-nowrap px-4 py-3 align-middle', render: item => <StatusValueBadge value={item.enabled ? 'enabled' : 'disabled'} /> },
          { key: 'actions', header: t('common.actions'), className: 'w-[1%] whitespace-nowrap px-4 py-3 text-right align-middle', render: item => (
            <div className="flex justify-end gap-2">
              <EditActionButton label={t('common.edit')} onClick={() => openDialog(item)} />
              <Button size="sm" variant="ghost" onClick={() => setSetToDelete(item)}>
                <Trash2 className="size-4" />
                {t('common.delete')}
              </Button>
            </div>
          ) },
        ]}
        emptyTitle={t('runtimeConfigSets.emptyTitle')}
        items={configSets.data ?? []}
        rowKey={item => item.id}
        variant="plain"
      />
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-h-[88vh] max-w-3xl overflow-hidden p-0">
          <DialogHeader className="border-b border-border px-6 py-5">
            <DialogTitle>{editingSet ? t('runtimeConfigSets.editTitle') : t('runtimeConfigSets.createTitle')}</DialogTitle>
            <DialogDescription>{t('runtimeConfigSets.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid max-h-[calc(88vh-96px)] grid-rows-[minmax(0,1fr)_auto]" onSubmit={form.handleSubmit(values => saveConfigSet.mutate(values))}>
            <div className="grid gap-4 overflow-y-auto px-6 py-5">
              <Field label={t('common.name')} required><Input {...form.register('name', { required: true })} /></Field>
              <Field hint={t('runtimeConfigSets.envVarsHint')} label={t('runtimeConfigSets.envVars')}>
                <Textarea className="min-h-24 font-mono text-sm" {...form.register('envVars')} placeholder={t('runtimeConfigSets.envVarsPlaceholder')} />
              </Field>
              <Field hint={t('runtimeConfigSets.configFilesHint')} label={t('runtimeConfigSets.configFiles')}>
                <RuntimeConfigFilesEditor
                  key={`${editingSet?.id ?? 'new'}-config-files`}
                  initialValue={form.getValues('configFiles') ?? ''}
                  onChange={value => form.setValue('configFiles', value, { shouldDirty: true, shouldValidate: true })}
                  onValidationChange={setConfigFilesValid}
                />
              </Field>
              <Field hint={editingSet?.secretRefsSet ? t('runtimeConfigSets.secretRefsConfiguredHint') : t('runtimeConfigSets.secretRefsHint')} label={t('runtimeConfigSets.secretRefs')}>
                <Textarea className="min-h-24 font-mono text-sm" {...form.register('secretRefs')} placeholder={t('runtimeConfigSets.secretRefsPlaceholder')} />
              </Field>
              <Field hint={editingSet?.secretFilesSet ? t('runtimeConfigSets.secretFilesConfiguredHint') : t('runtimeConfigSets.secretFilesHint')} label={t('runtimeConfigSets.secretFiles')}>
                <RuntimeConfigFilesEditor
                  key={`${editingSet?.id ?? 'new'}-secret-files`}
                  initialValue={form.getValues('secretFiles') ?? ''}
                  onChange={value => form.setValue('secretFiles', value, { shouldDirty: true, shouldValidate: true })}
                  onValidationChange={setSecretFilesValid}
                />
              </Field>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input className="size-4 accent-primary" type="checkbox" {...form.register('enabled')} />
                {t('common.enabled')}
              </label>
            </div>
            <DialogFooter className="border-t border-border bg-background px-6 py-4">
              <Button disabled={!configFilesValid || !secretFilesValid || saveConfigSet.isPending} type="submit">
                <FileCode2 className="size-4" />
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        cancelText={t('common.cancel')}
        confirmText={t('common.delete')}
        description={t('runtimeConfigSets.deleteDescription')}
        open={Boolean(setToDelete)}
        title={t('runtimeConfigSets.deleteTitle')}
        onConfirm={() => setToDelete && deleteConfigSet.mutate(setToDelete)}
        onOpenChange={open => !open && setSetToDelete(null)}
      />
    </Card>
  )
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
