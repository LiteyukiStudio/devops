import type { GitAccount, GitProvider, RepositoryBinding } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { GitBranch, LinkIcon, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'
import { api } from '@/api/client'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { GitRepositoryPicker } from '@/components/common/git-repository-picker'
import { PageHeader } from '@/components/common/page-header'
import { SearchSelect } from '@/components/common/search-select'
import { StatusBadge, StatusValueBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'

const bindingSchema = z.object({
  applicationId: z.string().min(1, i18next.t('repositories.applicationRequired')),
  gitAccountId: z.string().min(1, i18next.t('repositories.gitAccountRequired')),
  owner: z.string().min(1, i18next.t('repositories.ownerRequired')),
  repo: z.string().min(1, i18next.t('repositories.repoRequired')),
  cloneUrl: z.string().optional(),
  defaultBranch: z.string().optional(),
  webhookStatus: z.enum(['pending', 'created', 'disabled', 'failed']),
  autoConfigureWebhook: z.boolean().default(true),
})

type BindingFormInput = z.input<typeof bindingSchema>
type BindingForm = z.output<typeof bindingSchema>

export function RepositoryBindingsPage({ embedded = false, projectId: projectIdProp }: { embedded?: boolean, projectId?: string } = {}) {
  const { t } = useTranslation()
  const { projectId: routeProjectId = '' } = useParams()
  const projectId = projectIdProp ?? routeProjectId
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingBinding, setEditingBinding] = useState<RepositoryBinding | null>(null)
  const [bindingToDelete, setBindingToDelete] = useState<RepositoryBinding | null>(null)
  const [branchSearch, setBranchSearch] = useState('')
  const providers = useQuery({ queryKey: ['git-providers'], queryFn: () => api.listGitProviders() })
  const accounts = useQuery({ queryKey: ['git-accounts'], queryFn: () => api.listGitAccounts() })
  const applications = useQuery({
    queryKey: ['applications', projectId],
    queryFn: () => api.listApplications(projectId),
    enabled: Boolean(projectId),
  })
  const bindings = useQuery({
    queryKey: ['repository-bindings', projectId],
    queryFn: () => api.listRepositoryBindings(projectId),
    enabled: Boolean(projectId),
  })

  const form = useForm<BindingFormInput, undefined, BindingForm>({
    resolver: zodResolver(bindingSchema),
    mode: 'onChange',
    defaultValues: {
      applicationId: '',
      gitAccountId: '',
      owner: '',
      repo: '',
      cloneUrl: '',
      defaultBranch: 'main',
      webhookStatus: 'pending',
      autoConfigureWebhook: true,
    },
  })
  const selectedAccountId = form.watch('gitAccountId')
  const selectedOwner = form.watch('owner')
  const selectedRepo = form.watch('repo')
  const branches = useQuery({
    queryKey: ['git-branches', selectedAccountId, selectedOwner, selectedRepo, branchSearch],
    queryFn: () => api.listGitBranches(selectedAccountId || '', selectedOwner || '', selectedRepo || '', { search: branchSearch, limit: 50 }),
    enabled: Boolean(selectedAccountId && selectedOwner && selectedRepo),
  })

  const createBinding = useMutation({
    mutationFn: (payload: BindingForm) => api.createRepositoryBinding(projectId, {
      applicationId: payload.applicationId,
      gitAccountId: payload.gitAccountId,
      owner: payload.owner,
      repo: payload.repo,
      cloneUrl: payload.cloneUrl ?? '',
      defaultBranch: payload.defaultBranch ?? 'main',
      webhookStatus: payload.webhookStatus,
      autoConfigureWebhook: payload.autoConfigureWebhook,
    }),
    onSuccess: () => {
      toast.success(t('repositories.bindingSaved'))
      resetBindingForm()
      setDialogOpen(false)
      queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
      queryClient.invalidateQueries({ queryKey: ['applications', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  const updateBinding = useMutation({
    mutationFn: (payload: BindingForm) => {
      if (!editingBinding)
        throw new Error(t('repositories.editMissing'))
      return api.updateRepositoryBinding(projectId, editingBinding.id, {
        applicationId: payload.applicationId,
        gitAccountId: payload.gitAccountId,
        owner: payload.owner,
        repo: payload.repo,
        cloneUrl: payload.cloneUrl ?? '',
        defaultBranch: payload.defaultBranch ?? 'main',
        webhookStatus: payload.webhookStatus,
        autoConfigureWebhook: payload.autoConfigureWebhook,
      })
    },
    onSuccess: () => {
      toast.success(t('repositories.bindingSaved'))
      resetBindingForm()
      setEditingBinding(null)
      setDialogOpen(false)
      queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
      queryClient.invalidateQueries({ queryKey: ['applications', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteBinding = useMutation({
    mutationFn: (bindingId: string) => api.deleteRepositoryBinding(projectId, bindingId),
    onSuccess: () => {
      toast.success(t('repositories.bindingDeleted'))
      setBindingToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  const createWebhook = useMutation({
    mutationFn: (bindingId: string) => api.createRepositoryWebhook(projectId, bindingId),
    onSuccess: () => {
      toast.success(t('repositories.webhookCreated'))
      queryClient.invalidateQueries({ queryKey: ['repository-bindings', projectId] })
    },
    onError: error => toast.error(error.message),
  })

  function resetBindingForm(binding?: RepositoryBinding | null) {
    form.reset({
      applicationId: binding?.applicationId ?? '',
      gitAccountId: binding?.gitAccountId ?? '',
      owner: binding?.owner ?? '',
      repo: binding?.repo ?? '',
      cloneUrl: binding?.cloneUrl ?? '',
      defaultBranch: binding?.defaultBranch ?? 'main',
      webhookStatus: binding?.webhookStatus ?? 'pending',
      autoConfigureWebhook: true,
    })
    setBranchSearch('')
  }

  const openCreateDialog = () => {
    setEditingBinding(null)
    resetBindingForm()
    setDialogOpen(true)
  }

  const openEditDialog = (binding: RepositoryBinding) => {
    setEditingBinding(binding)
    resetBindingForm(binding)
    setDialogOpen(true)
  }

  return (
    <div className="grid gap-6">
      <PageHeader
        actions={!embedded
          ? (
              <div className="flex items-center gap-3">
                <Button type="button" onClick={openCreateDialog}>
                  <Plus size={16} />
                  {t('repositories.bindRepoTitle')}
                </Button>
                <Link className="text-sm text-primary hover:underline" to={`/projects/${projectId}/apps`}>{t('backToApps')}</Link>
              </div>
            )
          : undefined}
        description={t('repositories.description')}
        title={t('repositories.title')}
      />

      {(providers.isError || accounts.isError || bindings.isError) && (
        <ErrorState title={t('repositories.loadFailedTitle')} description={t('repositories.loadFailedDescription')} />
      )}

      <DataList
        columns={[
          {
            key: 'repo',
            header: t('repositories.repositorySearch'),
            render: binding => (
              <div className="flex min-w-0 items-center gap-3">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                  <GitBranch size={18} />
                </span>
                <div className="min-w-0">
                  <div className="truncate font-medium">{`${binding.owner}/${binding.repo}`}</div>
                  <p className="truncate text-sm text-muted-foreground">{binding.cloneUrl}</p>
                </div>
              </div>
            ),
          },
          { key: 'application', header: t('repositories.application'), render: binding => binding.applicationName ?? binding.applicationId },
          { key: 'branch', header: t('repositories.defaultBranch'), render: binding => <StatusBadge>{binding.defaultBranch}</StatusBadge> },
          { key: 'provider', header: t('codeRepositoriesView.provider'), render: binding => providerBindingLabel(binding, providers.data ?? [], accounts.data ?? []) },
          { key: 'webhook', header: t('repositories.webhookStatus'), render: binding => <StatusValueBadge value={binding.webhookStatus} /> },
          {
            key: 'actions',
            header: t('common.actions'),
            className: 'text-right whitespace-nowrap',
            render: binding => (
              <div className="flex justify-end gap-2">
                <Button disabled={createWebhook.isPending || binding.webhookStatus === 'created'} type="button" variant="secondary" onClick={() => createWebhook.mutate(binding.id)}>
                  <LinkIcon size={16} />
                  {t('repositories.createWebhook')}
                </Button>
                <EditActionButton aria-label={t('repositories.editAria')} label={t('edit')} onClick={() => openEditDialog(binding)} />
                <Button aria-label={t('repositories.deleteAria')} variant="ghost" onClick={() => setBindingToDelete(binding)}>
                  <Trash2 size={16} />
                </Button>
              </div>
            ),
          },
        ]}
        emptyTitle={t('repositories.emptyTitle')}
        emptyDescription={t('repositories.emptyDescription')}
        items={bindings.data ?? []}
        rowKey={binding => binding.id}
      />

      <ConfirmDialog
        confirmText={t('repositories.deleteConfirm')}
        description={t('repositories.deleteDescription', { repo: `${bindingToDelete?.owner ?? ''}/${bindingToDelete?.repo ?? ''}` })}
        open={Boolean(bindingToDelete)}
        pending={deleteBinding.isPending}
        title={t('repositories.deleteTitle')}
        onConfirm={() => bindingToDelete && deleteBinding.mutate(bindingToDelete.id)}
        onOpenChange={open => !open && setBindingToDelete(null)}
      />

      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) {
            setEditingBinding(null)
            resetBindingForm()
          }
        }}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{editingBinding ? t('repositories.editBinding') : t('repositories.bindRepoTitle')}</DialogTitle>
            <DialogDescription>{t('repositories.bindingDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-3"
            onSubmit={form.handleSubmit((values) => {
              if (editingBinding)
                updateBinding.mutate(values)
              else
                createBinding.mutate(values)
            })}
          >
            <div className="flex justify-end">
              <Link className="text-sm text-primary hover:underline" to="/code-repositories">{t('repositories.manageCodeRepositories')}</Link>
            </div>
            <Field error={form.formState.errors.applicationId?.message} label={t('repositories.application')} required>
              <Select {...form.register('applicationId')} aria-invalid={Boolean(form.formState.errors.applicationId)}>
                <option value="">{t('repositories.selectApplication')}</option>
                {(applications.data ?? []).map(application => (
                  <option key={application.id} value={application.id}>{application.name}</option>
                ))}
              </Select>
            </Field>
            <GitRepositoryPicker
              accounts={accounts.data ?? []}
              providers={providers.data ?? []}
              value={{
                gitAccountId: form.watch('gitAccountId') || '',
                owner: form.watch('owner') || '',
                repo: form.watch('repo') || '',
                cloneUrl: form.watch('cloneUrl') || '',
                defaultBranch: form.watch('defaultBranch') || 'main',
              }}
              onChange={(next) => {
                form.setValue('gitAccountId', next.gitAccountId, { shouldDirty: true, shouldValidate: true })
                form.setValue('owner', next.owner, { shouldDirty: true, shouldValidate: true })
                form.setValue('repo', next.repo, { shouldDirty: true, shouldValidate: true })
                form.setValue('cloneUrl', next.cloneUrl, { shouldDirty: true, shouldValidate: true })
                form.setValue('defaultBranch', next.defaultBranch || 'main', { shouldDirty: true, shouldValidate: true })
                setBranchSearch('')
              }}
            />
            <div className="grid gap-3 md:grid-cols-3">
              <Field error={form.formState.errors.owner?.message} label={t('repositories.owner')} required><Input {...form.register('owner')} aria-invalid={Boolean(form.formState.errors.owner)} placeholder={t('repositories.ownerPlaceholder')} /></Field>
              <Field error={form.formState.errors.repo?.message} label={t('repositories.repo')} required><Input {...form.register('repo')} aria-invalid={Boolean(form.formState.errors.repo)} placeholder={t('repositories.repoPlaceholder')} /></Field>
              <Field error={form.formState.errors.defaultBranch?.message} label={t('repositories.defaultBranch')}>
                <SearchSelect
                  disabled={!selectedAccountId || !selectedOwner || !selectedRepo}
                  emptyLabel={t('repositories.noBranches')}
                  limited={branches.data?.limited}
                  loading={branches.isFetching}
                  options={branchOptions(branches.data?.items ?? [], form.watch('defaultBranch'))}
                  placeholder={t('repositories.defaultBranchPlaceholder')}
                  search={branchSearch}
                  value={form.watch('defaultBranch') || ''}
                  onSearchChange={setBranchSearch}
                  onValueChange={value => form.setValue('defaultBranch', value, { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <Field error={form.formState.errors.cloneUrl?.message} label={t('repositories.cloneUrl')}><Input {...form.register('cloneUrl')} aria-invalid={Boolean(form.formState.errors.cloneUrl)} placeholder={t('repositories.cloneUrlPlaceholder')} /></Field>
              <label className="flex items-start gap-3 rounded-md border border-border bg-muted/30 p-3">
                <input className="mt-1 size-4 accent-primary" type="checkbox" {...form.register('autoConfigureWebhook')} />
                <span className="grid gap-1 text-sm">
                  <span className="font-medium text-foreground">{t('repositories.autoConfigureWebhook')}</span>
                  <span className="text-xs leading-5 text-muted-foreground">{t('repositories.autoConfigureWebhookHint')}</span>
                </span>
              </label>
            </div>
            <DialogFooter>
              <Button disabled={createBinding.isPending || updateBinding.isPending || (accounts.data ?? []).length === 0 || !form.formState.isValid} type="submit">
                <Plus size={16} />
                {t('repositories.saveBinding')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function providerBindingLabel(binding: RepositoryBinding, providers: GitProvider[], accounts: GitAccount[]) {
  const provider = providers.find(item => item.id === binding.gitProviderId)
  const account = accounts.find(item => item.id === binding.gitAccountId)

  return (
    <span className="block max-w-64 truncate text-sm text-muted-foreground">
      {provider?.name ?? binding.providerName ?? binding.gitProviderId}
      {' · '}
      {account?.username ?? binding.accountUsername ?? binding.gitAccountId}
      {binding.accountOwnerEmail ? ` · ${binding.accountOwnerName || binding.accountOwnerEmail}` : ''}
    </span>
  )
}

function branchOptions(branches: Array<{ name: string }>, current?: string) {
  const options = branches.map(branch => ({ value: branch.name, label: branch.name }))
  const normalized = current?.trim()
  if (normalized && !options.some(option => option.value === normalized))
    options.unshift({ value: normalized, label: normalized })
  return options
}
