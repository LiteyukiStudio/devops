import type { GitAccount, GitProvider } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { Copy, ExternalLink, Info, KeyRound, LinkIcon, Plus, RefreshCw, Save, Trash2 } from 'lucide-react'
import { motion } from 'motion/react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'
import { api, apiBaseOrigin, gitOAuthStartUrl } from '@/api/client'
import { useSession } from '@/app/session-context'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ContentTabs } from '@/components/common/content-tabs'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { ProjectSpaceMultiSelect } from '@/components/common/project-space-select'
import { StatusBadge, StatusValueBadge } from '@/components/common/status-badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { TabsContent } from '@/components/ui/tabs'

const providerSchema = z.object({
  name: z.string().min(1, i18next.t('codeRepositoriesView.providerNameRequired')),
  type: z.enum(['github', 'gitea', 'gitlab']),
  baseUrl: z.string().optional(),
  scope: z.enum(['global', 'project', 'user']),
  ownerRef: z.string(),
  projectIds: z.array(z.string()),
  authType: z.enum(['oauth', 'pat']),
  clientId: z.string().optional(),
  clientSecret: z.string().optional(),
  enabled: z.boolean(),
}).superRefine((value, ctx) => {
  if (value.scope === 'project' && value.projectIds.length === 0) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['projectIds'],
      message: i18next.t('codeRepositoriesView.ownerProjectRequired'),
    })
  }
})

const credentialSchema = z.object({
  providerId: z.string().min(1, i18next.t('codeRepositoriesView.providerRequired')),
  scope: z.enum(['global', 'project', 'user']),
  ownerRef: z.string(),
  projectIds: z.array(z.string()),
  username: z.string().min(1, i18next.t('codeRepositoriesView.usernameRequired')),
  externalUserId: z.string().optional(),
  avatarUrl: z.string().optional(),
  accessToken: z.string().optional(),
  refreshToken: z.string().optional(),
  scopesText: z.string().optional(),
  accessScope: z.enum(['personal', 'provider']),
  status: z.enum(['connected', 'expired', 'revoked']),
}).superRefine((value, ctx) => {
  if (value.scope === 'project' && value.projectIds.length === 0) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['projectIds'],
      message: i18next.t('codeRepositoriesView.ownerProjectRequired'),
    })
  }
})

type ProviderForm = z.infer<typeof providerSchema>
type CredentialForm = z.infer<typeof credentialSchema>

const providerDefaults: ProviderForm = {
  authType: 'oauth',
  baseUrl: 'https://github.com',
  ownerRef: '',
  projectIds: [],
  clientId: '',
  clientSecret: '',
  scope: 'global',
  enabled: true,
  name: '',
  type: 'github',
}

const credentialDefaults: CredentialForm = {
  accessScope: 'personal',
  accessToken: '',
  avatarUrl: '',
  externalUserId: '',
  ownerRef: '',
  projectIds: [],
  providerId: '',
  refreshToken: '',
  scope: 'user',
  scopesText: 'repo,read:user',
  status: 'connected',
  username: '',
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

export function CodeRepositoriesPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { user } = useSession()
  const [activeTab, setActiveTab] = useState('providers')
  const [providerDialogOpen, setProviderDialogOpen] = useState(false)
  const [credentialDialogOpen, setCredentialDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<GitProvider | null>(null)
  const [providerToDelete, setProviderToDelete] = useState<GitProvider | null>(null)
  const [credentialToDelete, setCredentialToDelete] = useState<GitAccount | null>(null)
  const [providerPage, setProviderPage] = useState(1)
  const [providerPageSize, setProviderPageSize] = useState(10)
  const [credentialPage, setCredentialPage] = useState(1)
  const [credentialPageSize, setCredentialPageSize] = useState(10)
  const providers = useQuery({
    queryKey: ['git-providers', providerPage, providerPageSize],
    queryFn: () => api.listGitProvidersPage({ page: providerPage, pageSize: providerPageSize, sortBy: 'createdAt', sortOrder: 'desc' }),
  })
  const credentials = useQuery({
    queryKey: ['git-accounts', credentialPage, credentialPageSize],
    queryFn: () => api.listGitAccountsPage({ page: credentialPage, pageSize: credentialPageSize, sortBy: 'createdAt', sortOrder: 'desc' }),
  })
  const providerOptions = useQuery({ queryKey: ['git-providers', 'options'], queryFn: () => api.listGitProviders() })
  const providerItems = useMemo(() => providers.data?.items ?? [], [providers.data?.items])
  const providerOptionItems = useMemo(() => providerOptions.data ?? [], [providerOptions.data])
  const credentialItems = useMemo(() => credentials.data?.items ?? [], [credentials.data?.items])
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
  const canManageProviders = user?.permissions.includes('user.manage')
  const projectMap = useMemo(() => {
    const map: Record<string, string> = {}
    for (const project of projects.data ?? [])
      map[project.id] = project.name
    return map
  }, [projects.data])

  const providerForm = useForm<ProviderForm>({
    resolver: zodResolver(providerSchema),
    mode: 'onChange',
    defaultValues: providerDefaults,
  })
  const credentialForm = useForm<CredentialForm>({
    resolver: zodResolver(credentialSchema),
    mode: 'onChange',
    defaultValues: credentialDefaults,
  })
  const providerType = providerForm.watch('type')
  const providerBaseUrl = providerForm.watch('baseUrl')
  const providerName = providerForm.watch('name')
  const providerAuthType = providerForm.watch('authType')
  const selectedCredentialProviderId = credentialForm.watch('providerId')
  const selectedCredentialProvider = providerOptionItems.find(provider => provider.id === selectedCredentialProviderId)
  const providerGuide = gitProviderGuide(providerType, providerBaseUrl, providerName)
  const isGithubProvider = providerType === 'github'
  const providerScope = providerForm.watch('scope')
  const credentialScope = credentialForm.watch('scope')
  const hasGithubProvider = providerOptionItems.some(provider => provider.type === 'github')
  const hasAnotherGithubProvider = useMemo(() => {
    if (!editingProvider)
      return hasGithubProvider
    return providerOptionItems.some(provider => provider.type === 'github' && provider.id !== editingProvider.id)
  }, [editingProvider, hasGithubProvider, providerOptionItems])

  useEffect(() => {
    if (!editingProvider) {
      providerForm.reset(providerDefaults)
      return
    }
    providerForm.reset({
      authType: editingProvider.authType === 'pat' ? 'pat' : 'oauth',
      baseUrl: editingProvider.baseUrl,
      scope: editingProvider.scope ?? 'user',
      ownerRef: editingProvider.ownerRef,
      projectIds: editingProvider.projectIds ?? [],
      clientId: editingProvider.clientId,
      clientSecret: '',
      enabled: editingProvider.enabled,
      name: editingProvider.name,
      type: editingProvider.type,
    })
  }, [editingProvider, providerForm])

  useEffect(() => {
    if (isGithubProvider) {
      providerForm.setValue('baseUrl', normalizeGitBaseUrl('github'), { shouldDirty: true, shouldValidate: true })
      providerForm.setValue('scope', 'global', { shouldDirty: true, shouldValidate: true })
      providerForm.setValue('ownerRef', '', { shouldDirty: true, shouldValidate: true })
      providerForm.setValue('projectIds', [], { shouldDirty: true, shouldValidate: true })
    }
  }, [isGithubProvider, providerForm])

  useEffect(() => {
    if (providerScope !== 'project')
      providerForm.setValue('projectIds', [])
  }, [providerScope, providerForm])

  useEffect(() => {
    if (credentialScope !== 'project')
      credentialForm.setValue('projectIds', [])
  }, [credentialScope, credentialForm])

  const saveProvider = useMutation({
    mutationFn: (payload: ProviderForm) => {
      const providerPayload = {
        authType: payload.authType,
        baseUrl: normalizeGitBaseUrl(payload.type, payload.baseUrl),
        clientId: payload.clientId ?? '',
        clientSecret: payload.clientSecret ?? '',
        enabled: payload.enabled,
        name: payload.name,
        scope: payload.scope,
        ownerRef: '',
        projectIds: payload.scope === 'project' ? payload.projectIds : [],
        type: payload.type,
      }
      if (editingProvider)
        return api.updateGitProvider(editingProvider.id, providerPayload)
      return api.createGitProvider(providerPayload)
    },
    onSuccess: () => {
      toast.success(t(editingProvider ? 'codeRepositoriesView.providerUpdated' : 'codeRepositoriesView.providerCreated'))
      setProviderDialogOpen(false)
      setEditingProvider(null)
      providerForm.reset(providerDefaults)
      queryClient.invalidateQueries({ queryKey: ['git-providers'] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteProvider = useMutation({
    mutationFn: api.deleteGitProvider,
    onSuccess: () => {
      toast.success(t('codeRepositoriesView.providerDeleted'))
      setProviderToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['git-providers'] })
    },
    onError: error => toast.error(error.message),
  })

  const createCredential = useMutation({
    mutationFn: (payload: CredentialForm) => api.createGitAccount({
      accessScope: payload.accessScope,
      accessToken: payload.accessToken ?? '',
      avatarUrl: payload.avatarUrl ?? '',
      ownerRef: '',
      projectIds: payload.scope === 'project' ? payload.projectIds : [],
      externalUserId: payload.externalUserId ?? '',
      providerId: payload.providerId,
      refreshToken: payload.refreshToken ?? '',
      scope: payload.scope,
      scopes: splitText(payload.scopesText),
      status: payload.status,
      username: payload.username,
    }),
    onSuccess: () => {
      toast.success(t('codeRepositoriesView.credentialCreated'))
      setCredentialDialogOpen(false)
      credentialForm.reset(credentialDefaults)
      queryClient.invalidateQueries({ queryKey: ['git-accounts'] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteCredential = useMutation({
    mutationFn: api.deleteGitAccount,
    onSuccess: () => {
      toast.success(t('codeRepositoriesView.credentialDeleted'))
      setCredentialToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['git-accounts'] })
    },
    onError: error => toast.error(error.message),
  })

  const refreshCredential = useMutation({
    mutationFn: api.refreshGitAccount,
    onSuccess: () => {
      toast.success(t('codeRepositoriesView.credentialReloaded'))
      queryClient.invalidateQueries({ queryKey: ['git-accounts'] })
    },
    onError: error => toast.error(error.message),
  })

  return (
    <div className="grid gap-6">
      {(providers.isError || credentials.isError) && (
        <ErrorState title={t('codeRepositoriesView.loadFailedTitle')} description={t('codeRepositoriesView.loadFailedDescription')} />
      )}

      <ContentTabs
        tabs={[
          { value: 'providers', label: t('codeRepositoriesView.providersTab') },
          { value: 'credentials', label: t('codeRepositoriesView.credentialsTab') },
        ]}
        tools={(
          activeTab === 'providers'
            ? (
                canManageProviders
                  ? (
                      <Button
                        onClick={() => {
                          setEditingProvider(null)
                          providerForm.reset(providerDefaults)
                          setProviderDialogOpen(true)
                        }}
                      >
                        <Plus size={16} />
                        {t('codeRepositoriesView.createProvider')}
                      </Button>
                    )
                  : undefined
              )
            : (
                <Button onClick={() => setCredentialDialogOpen(true)}>
                  <Plus size={16} />
                  {t('codeRepositoriesView.createCredential')}
                </Button>
              )
        )}
        value={activeTab}
        onValueChange={setActiveTab}
      >
        <TabsContent value={activeTab}>
          <motion.div
            key={activeTab}
            animate={{ opacity: 1, y: 0 }}
            initial={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
          >
            {activeTab === 'providers'
              ? (
                  <ProvidersPanel
                    canManage={Boolean(canManageProviders)}
                    page={providers.data?.page ?? providerPage}
                    pageSize={providers.data?.pageSize ?? providerPageSize}
                    providers={providerOptionItems}
                    projectMap={projectMap}
                    total={providers.data?.total ?? 0}
                    totalPages={providers.data?.totalPages ?? 0}
                    onDelete={setProviderToDelete}
                    onEdit={(provider) => {
                      setEditingProvider(provider)
                      setProviderDialogOpen(true)
                    }}
                    onPageChange={setProviderPage}
                    onPageSizeChange={(pageSize) => {
                      setProviderPageSize(pageSize)
                      setProviderPage(1)
                    }}
                  />
                )
              : (
                  <CredentialsPanel
                    credentials={credentialItems}
                    page={credentials.data?.page ?? credentialPage}
                    pageSize={credentials.data?.pageSize ?? credentialPageSize}
                    providers={providerItems}
                    projectMap={projectMap}
                    refreshPending={refreshCredential.isPending}
                    total={credentials.data?.total ?? 0}
                    totalPages={credentials.data?.totalPages ?? 0}
                    onDelete={setCredentialToDelete}
                    onPageChange={setCredentialPage}
                    onPageSizeChange={(pageSize) => {
                      setCredentialPageSize(pageSize)
                      setCredentialPage(1)
                    }}
                    onRefresh={credential => refreshCredential.mutate(credential.id)}
                  />
                )}
          </motion.div>
        </TabsContent>
      </ContentTabs>

      <Dialog
        open={providerDialogOpen}
        onOpenChange={(open) => {
          setProviderDialogOpen(open)
          if (!open)
            setEditingProvider(null)
        }}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{editingProvider ? t('codeRepositoriesView.editProvider') : t('codeRepositoriesView.createProvider')}</DialogTitle>
            <DialogDescription>{t('codeRepositoriesView.providerDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={providerForm.handleSubmit(values => saveProvider.mutate(values))}>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={providerForm.formState.errors.name?.message} hint={t('codeRepositoriesView.providerNameHint')} label={t('codeRepositoriesView.name')} required><Input {...providerForm.register('name')} aria-invalid={Boolean(providerForm.formState.errors.name)} placeholder={t('codeRepositoriesView.providerNamePlaceholder')} /></Field>
              <Field error={providerForm.formState.errors.type?.message} hint={t('codeRepositoriesView.providerTypeHint')} label={t('codeRepositoriesView.type')} required>
                <div className="flex gap-2">
                  <GitProviderIcon baseUrl={providerBaseUrl} type={providerType} />
                  <Select {...providerForm.register('type')} aria-invalid={Boolean(providerForm.formState.errors.type)}>
                    <option value="github" disabled={hasAnotherGithubProvider && (!editingProvider || editingProvider.type !== 'github')}>{t('codeRepositoriesView.github')}</option>
                    <option value="gitea">{t('codeRepositoriesView.gitea')}</option>
                    <option value="gitlab">{t('codeRepositoriesView.gitlab')}</option>
                  </Select>
                </div>
              </Field>
            </div>
            {hasAnotherGithubProvider && !editingProvider?.type && (
              <Alert>
                <Info />
                <AlertDescription>{t('codeRepositoriesView.githubProviderOnlyOne')}</AlertDescription>
              </Alert>
            )}
            <Field error={providerForm.formState.errors.baseUrl?.message} hint={t('codeRepositoriesView.baseUrlHint')} label={t('codeRepositoriesView.baseUrl')}>
              <Input
                {...providerForm.register('baseUrl')}
                aria-invalid={Boolean(providerForm.formState.errors.baseUrl)}
                disabled={isGithubProvider}
                placeholder={t('codeRepositoriesView.baseUrlPlaceholder')}
              />
            </Field>
            <Field error={providerForm.formState.errors.scope?.message} hint={t('codeRepositoriesView.scopeHint')} label={t('codeRepositoriesView.scope')} required>
              <Select {...providerForm.register('scope')} aria-invalid={Boolean(providerForm.formState.errors.scope)} disabled={isGithubProvider}>
                <option value="global">{t('codeRepositoriesView.scopeGlobal')}</option>
                <option value="project">{t('codeRepositoriesView.scopeProject')}</option>
                <option value="user">{t('codeRepositoriesView.scopeUser')}</option>
              </Select>
            </Field>
            {providerScope === 'project' && (
              <Field error={providerForm.formState.errors.projectIds?.message} hint={t('codeRepositoriesView.ownerProjectHint')} label={t('codeRepositoriesView.ownerProject')} required>
                <ProjectSpaceMultiSelect
                  projects={projects.data ?? []}
                  value={providerForm.watch('projectIds')}
                  onChange={value => providerForm.setValue('projectIds', value, { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            )}
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={providerForm.formState.errors.authType?.message} hint={t('codeRepositoriesView.authTypeHint')} label={t('codeRepositoriesView.authType')} required>
                <Select {...providerForm.register('authType')} aria-invalid={Boolean(providerForm.formState.errors.authType)}>
                  <option value="oauth">{t('codeRepositoriesView.oauth')}</option>
                  <option value="pat">{t('codeRepositoriesView.pat')}</option>
                </Select>
              </Field>
              <Field error={providerForm.formState.errors.clientId?.message} hint={t('codeRepositoriesView.clientIdHint')} label={t('codeRepositoriesView.clientId')}><Input {...providerForm.register('clientId')} aria-invalid={Boolean(providerForm.formState.errors.clientId)} /></Field>
            </div>
            <Field error={providerForm.formState.errors.clientSecret?.message} hint={t('codeRepositoriesView.clientSecretHint')} label={t('codeRepositoriesView.clientSecret')}>
              <Input
                {...providerForm.register('clientSecret')}
                aria-invalid={Boolean(providerForm.formState.errors.clientSecret)}
                placeholder={editingProvider?.clientSecretSet ? t('codeRepositoriesView.secretSetPlaceholder') : t('codeRepositoriesView.clientSecretPlaceholder')}
                type="password"
              />
            </Field>
            {providerAuthType === 'oauth' && <OAuthAppGuide guide={providerGuide} />}
            <DialogFooter>
              <Button disabled={saveProvider.isPending || !providerForm.formState.isValid} type="submit">
                {editingProvider ? <Save size={16} /> : <Plus size={16} />}
                {editingProvider ? t('codeRepositoriesView.saveProvider') : t('codeRepositoriesView.createProvider')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={credentialDialogOpen}
        onOpenChange={(open) => {
          setCredentialDialogOpen(open)
          if (!open)
            credentialForm.reset(credentialDefaults)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('codeRepositoriesView.createCredential')}</DialogTitle>
            <DialogDescription>{t('codeRepositoriesView.credentialDialogDescription')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={credentialForm.handleSubmit(values => createCredential.mutate(values))}>
            <Field error={credentialForm.formState.errors.providerId?.message} label={t('codeRepositoriesView.provider')} required>
              <Select {...credentialForm.register('providerId')} aria-invalid={Boolean(credentialForm.formState.errors.providerId)}>
                <option value="">{t('codeRepositoriesView.selectProvider')}</option>
                {providerOptionItems.map(provider => (
                  <option key={provider.id} value={provider.id}>{provider.name}</option>
                ))}
              </Select>
            </Field>
            {selectedCredentialProvider && <CredentialOAuthGuide provider={selectedCredentialProvider} />}
            <Field error={credentialForm.formState.errors.scope?.message} hint={t('codeRepositoriesView.scopeHint')} label={t('codeRepositoriesView.scope')} required>
              <Select {...credentialForm.register('scope')} aria-invalid={Boolean(credentialForm.formState.errors.scope)}>
                <option value="global">{t('codeRepositoriesView.scopeGlobal')}</option>
                <option value="project">{t('codeRepositoriesView.scopeProject')}</option>
                <option value="user">{t('codeRepositoriesView.scopeUser')}</option>
              </Select>
            </Field>
            {credentialScope === 'project' && (
              <Field error={credentialForm.formState.errors.projectIds?.message} hint={t('codeRepositoriesView.ownerProjectHint')} label={t('codeRepositoriesView.ownerProject')} required>
                <ProjectSpaceMultiSelect
                  projects={projects.data ?? []}
                  value={credentialForm.watch('projectIds')}
                  onChange={value => credentialForm.setValue('projectIds', value, { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            )}
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={credentialForm.formState.errors.username?.message} hint={t('codeRepositoriesView.usernameHint')} label={t('codeRepositoriesView.username')} required><Input {...credentialForm.register('username')} aria-invalid={Boolean(credentialForm.formState.errors.username)} placeholder={t('codeRepositoriesView.usernamePlaceholder')} /></Field>
              <Field error={credentialForm.formState.errors.accessScope?.message} hint={t('codeRepositoriesView.accessScopeHint')} label={t('codeRepositoriesView.accessScope')} required>
                <Select {...credentialForm.register('accessScope')} aria-invalid={Boolean(credentialForm.formState.errors.accessScope)}>
                  <option value="personal">{t('codeRepositoriesView.personalScope')}</option>
                  <option value="provider">{t('codeRepositoriesView.providerScope')}</option>
                </Select>
              </Field>
            </div>
            <Field error={credentialForm.formState.errors.accessToken?.message} hint={t('codeRepositoriesView.accessTokenHint')} label={t('codeRepositoriesView.accessToken')}>
              <Input {...credentialForm.register('accessToken')} aria-invalid={Boolean(credentialForm.formState.errors.accessToken)} type="password" />
            </Field>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={credentialForm.formState.errors.scopesText?.message} hint={t('codeRepositoriesView.scopesHint')} label={t('codeRepositoriesView.scopes')}>
                <Input {...credentialForm.register('scopesText')} aria-invalid={Boolean(credentialForm.formState.errors.scopesText)} />
              </Field>
              <Field error={credentialForm.formState.errors.status?.message} label={t('codeRepositoriesView.status')} required>
                <Select {...credentialForm.register('status')} aria-invalid={Boolean(credentialForm.formState.errors.status)}>
                  <option value="connected">{t('common.connected')}</option>
                  <option value="expired">{t('common.expired')}</option>
                  <option value="revoked">{t('common.revoked')}</option>
                </Select>
              </Field>
            </div>
            <DialogFooter>
              <Button disabled={createCredential.isPending || !credentialForm.formState.isValid} type="submit">
                <Plus size={16} />
                {t('codeRepositoriesView.createCredential')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        confirmText={t('codeRepositoriesView.deleteProviderConfirm')}
        description={t('codeRepositoriesView.deleteProviderDescription', { name: providerToDelete?.name ?? '' })}
        open={Boolean(providerToDelete)}
        pending={deleteProvider.isPending}
        title={t('codeRepositoriesView.deleteProviderTitle')}
        onConfirm={() => providerToDelete && deleteProvider.mutate(providerToDelete.id)}
        onOpenChange={open => !open && setProviderToDelete(null)}
      />
      <ConfirmDialog
        confirmText={t('codeRepositoriesView.deleteCredentialConfirm')}
        description={t('codeRepositoriesView.deleteCredentialDescription', { name: credentialToDelete?.username ?? '' })}
        open={Boolean(credentialToDelete)}
        pending={deleteCredential.isPending}
        title={t('codeRepositoriesView.deleteCredentialTitle')}
        onConfirm={() => credentialToDelete && deleteCredential.mutate(credentialToDelete.id)}
        onOpenChange={open => !open && setCredentialToDelete(null)}
      />
    </div>
  )
}

function ProvidersPanel({
  canManage,
  page,
  pageSize,
  providers,
  projectMap,
  total,
  totalPages,
  onDelete,
  onEdit,
  onPageChange,
  onPageSizeChange,
}: {
  canManage: boolean
  page: number
  pageSize: number
  providers: GitProvider[]
  projectMap: Record<string, string>
  total: number
  totalPages: number
  onDelete: (provider: GitProvider) => void
  onEdit: (provider: GitProvider) => void
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}) {
  const { t } = useTranslation()
  return (
    <DataList
      columns={[
        {
          key: 'name',
          header: t('common.name'),
          render: provider => (
            <div className="flex min-w-0 items-center gap-3">
              <GitProviderIcon baseUrl={provider.baseUrl} type={provider.type} />
              <div className="min-w-0">
                <div className="truncate font-medium">{provider.name}</div>
                <p className="truncate text-sm text-muted-foreground">{provider.baseUrl}</p>
              </div>
            </div>
          ),
        },
        { key: 'type', header: t('common.type'), render: provider => <StatusBadge>{provider.type}</StatusBadge> },
        { key: 'auth', header: t('codeRepositoriesView.authType'), render: provider => <StatusBadge>{provider.authType}</StatusBadge> },
        {
          key: 'scope',
          header: t('common.scope'),
          render: provider => (
            <div className="flex flex-wrap gap-2">
              <StatusBadge>{provider.scope}</StatusBadge>
              {projectScopeBadges(provider.projectIds, projectMap)}
            </div>
          ),
        },
        { key: 'secret', header: t('codeRepositoriesView.clientSecret'), render: provider => provider.clientSecretSet ? t('codeRepositoriesView.secretSet') : t('codeRepositoriesView.secretNotSet') },
        { key: 'status', header: t('common.status'), render: provider => <StatusValueBadge value={provider.enabled ? 'enabled' : 'disabled'} /> },
        {
          key: 'actions',
          header: t('common.actions'),
          className: 'text-right whitespace-nowrap',
          render: provider => canManage
            ? (
                <div className="flex shrink-0 items-center gap-2">
                  <EditActionButton aria-label={t('edit')} label={t('edit')} onClick={() => onEdit(provider)} />
                  <Button aria-label={t('codeRepositoriesView.deleteProviderAria')} variant="ghost" onClick={() => onDelete(provider)}>
                    <Trash2 size={16} />
                  </Button>
                </div>
              )
            : <span className="text-xs text-muted-foreground">{t('common.viewOnly')}</span>,
        },
      ]}
      emptyTitle={t('codeRepositoriesView.noProvidersTitle')}
      emptyDescription={t('codeRepositoriesView.noProvidersDescription')}
      items={providers}
      pagination={{
        page,
        pageSize,
        pageSizeOptions: PAGE_SIZE_OPTIONS,
        total,
        totalPages,
        pageInfoLabel: t('pagination.pageInfo', { page, total, totalPages }),
        onPageChange,
        onPageSizeChange,
      }}
      rowKey={provider => provider.id}
    />
  )
}

function CredentialsPanel({
  credentials,
  page,
  pageSize,
  providers,
  projectMap,
  refreshPending,
  total,
  totalPages,
  onDelete,
  onPageChange,
  onPageSizeChange,
  onRefresh,
}: {
  credentials: GitAccount[]
  page: number
  pageSize: number
  providers: GitProvider[]
  projectMap: Record<string, string>
  refreshPending: boolean
  total: number
  totalPages: number
  onDelete: (credential: GitAccount) => void
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  onRefresh: (credential: GitAccount) => void
}) {
  const { t } = useTranslation()
  const oauthProviders = providers.filter(provider => isGitOAuthReady(provider))
  const oauthBlockedProviders = providers.filter(provider => provider.enabled && provider.authType === 'oauth' && !isGitOAuthReady(provider))
  return (
    <div className="grid gap-4">
      {oauthProviders.length > 0 && (
        <div className="grid gap-2">
          {oauthProviders.map(provider => (
            <Button key={provider.id} type="button" variant="secondary" onClick={() => { window.location.href = gitOAuthStartUrl(provider.id, '/code-repositories', window.location.origin) }}>
              <LinkIcon size={16} />
              {t('codeRepositoriesView.oauthConnect', { provider: provider.name })}
            </Button>
          ))}
        </div>
      )}
      {oauthBlockedProviders.length > 0 && (
        <Alert>
          <Info />
          <AlertTitle>{t('codeRepositoriesView.oauthUnavailableTitle')}</AlertTitle>
          <AlertDescription>
            {t('codeRepositoriesView.oauthUnavailableDescription', { providers: oauthBlockedProviders.map(provider => provider.name).join(', ') })}
          </AlertDescription>
        </Alert>
      )}
      <DataList
        columns={[
          {
            key: 'name',
            header: t('codeRepositoriesView.username'),
            render: credential => (
              <div className="flex min-w-0 items-center gap-3">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground"><KeyRound size={18} /></span>
                <div className="min-w-0">
                  <div className="truncate font-medium">{credential.username}</div>
                  <p className="truncate text-sm text-muted-foreground">{credential.scopes || t('codeRepositoriesView.noScopes')}</p>
                </div>
              </div>
            ),
          },
          { key: 'provider', header: t('codeRepositoriesView.provider'), render: credential => <ProviderNameBadge provider={providers.find(provider => provider.id === credential.providerId)} providerId={credential.providerId} /> },
          {
            key: 'scope',
            header: t('common.scope'),
            render: credential => (
              <div className="flex flex-wrap gap-2">
                <StatusBadge>{credential.accessScope === 'provider' ? t('codeRepositoriesView.providerScope') : t('codeRepositoriesView.personalScope')}</StatusBadge>
                <StatusBadge>{credential.scope}</StatusBadge>
                {projectScopeBadges(credential.projectIds, projectMap)}
              </div>
            ),
          },
          {
            key: 'tokens',
            header: t('codeRepositoriesView.accessToken'),
            render: credential => (
              <span className="text-sm text-muted-foreground">
                {credential.accessTokenSet ? t('codeRepositoriesView.accessTokenSet') : t('codeRepositoriesView.accessTokenNotSet')}
                {' · '}
                {credential.refreshTokenSet ? t('codeRepositoriesView.refreshTokenSet') : t('codeRepositoriesView.refreshTokenNotSet')}
              </span>
            ),
          },
          { key: 'status', header: t('common.status'), render: credential => <StatusValueBadge value={credential.status} /> },
          {
            key: 'actions',
            header: t('common.actions'),
            className: 'text-right whitespace-nowrap',
            render: credential => (
              <div className="flex justify-end gap-2">
                <Button disabled={refreshPending || !credential.refreshTokenSet} type="button" variant="ghost" onClick={() => onRefresh(credential)}>
                  <RefreshCw size={16} />
                  {t('codeRepositoriesView.refreshCredential')}
                </Button>
                <Button aria-label={t('codeRepositoriesView.deleteCredentialAria')} variant="ghost" onClick={() => onDelete(credential)}>
                  <Trash2 size={16} />
                </Button>
              </div>
            ),
          },
        ]}
        emptyTitle={t('codeRepositoriesView.noCredentialsTitle')}
        emptyDescription={t('codeRepositoriesView.noCredentialsDescription')}
        items={credentials}
        pagination={{
          page,
          pageSize,
          pageSizeOptions: PAGE_SIZE_OPTIONS,
          total,
          totalPages,
          pageInfoLabel: t('pagination.pageInfo', { page, total, totalPages }),
          onPageChange,
          onPageSizeChange,
        }}
        rowKey={credential => credential.id}
      />
    </div>
  )
}

function ProviderNameBadge({ provider, providerId }: { provider?: GitProvider, providerId: string }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1 rounded-full border border-border bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
      <GitProviderIcon baseUrl={provider?.baseUrl} className="size-4 rounded-sm border-0 bg-transparent p-0" type={provider?.type ?? 'github'} />
      <span className="truncate">{provider?.name ?? providerId}</span>
    </span>
  )
}

function GitProviderIcon({
  baseUrl,
  className,
  type,
}: {
  baseUrl?: string
  className?: string
  type: GitProvider['type']
}) {
  const faviconUrl = gitProviderFaviconUrl(type, baseUrl)

  return (
    <span className={className ?? 'flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted p-1 text-muted-foreground'}>
      {faviconUrl
        ? <GitProviderFavicon key={faviconUrl} faviconUrl={faviconUrl} type={type} />
        : <GitProviderFallbackIcon type={type} />}
    </span>
  )
}

function GitProviderFavicon({ faviconUrl, type }: { faviconUrl: string, type: GitProvider['type'] }) {
  const [faviconFailed, setFaviconFailed] = useState(false)

  if (faviconFailed)
    return <GitProviderFallbackIcon type={type} />

  return (
    <img
      alt=""
      className="size-full rounded-sm object-contain"
      src={faviconUrl}
      onError={() => setFaviconFailed(true)}
    />
  )
}

function GitProviderFallbackIcon({ type }: { type: GitProvider['type'] }) {
  if (type === 'gitea') {
    return (
      <svg aria-hidden="true" className="size-full" viewBox="0 0 24 24">
        <circle cx="12" cy="12" fill="#609926" r="11" />
        <path d="M7.2 9.1h8.1a3.2 3.2 0 0 1 0 6.4h-.9a4.4 4.4 0 0 1-8.5-1.6V10.4c0-.7.5-1.3 1.3-1.3Z" fill="#fff" />
        <path d="M14.8 11.2h1.1a1.1 1.1 0 1 1 0 2.2h-1.1Z" fill="#609926" />
        <path d="M8.2 11.5h5.4M8.2 13.4h4.2" stroke="#609926" strokeLinecap="round" strokeWidth="1.3" />
      </svg>
    )
  }
  if (type === 'gitlab') {
    return (
      <svg aria-hidden="true" className="size-full" viewBox="0 0 24 24">
        <path d="m12 21 4.2-12.9H7.8Z" fill="#E24329" />
        <path d="m12 21-8.6-6.2 4.2-12.2Z" fill="#FC6D26" />
        <path d="m12 21 8.6-6.2-4.2-12.2Z" fill="#FC6D26" />
        <path d="M3.4 14.8h8.6L7.6 2.6Z" fill="#FCA326" />
        <path d="M20.6 14.8H12l4.4-12.2Z" fill="#FCA326" />
      </svg>
    )
  }
  return (
    <svg aria-hidden="true" className="size-full" viewBox="0 0 24 24">
      <path
        d="M12 2.4a9.6 9.6 0 0 0-3 18.7c.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.2-3.4-1.2-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.3 1.1 2.9.8.1-.7.4-1.1.7-1.4-2.2-.3-4.5-1.1-4.5-4.7 0-1 .4-1.9 1-2.6-.1-.3-.4-1.3.1-2.6 0 0 .8-.3 2.7 1a9.3 9.3 0 0 1 4.9 0c1.9-1.3 2.7-1 2.7-1 .5 1.3.2 2.3.1 2.6.6.7 1 1.6 1 2.6 0 3.7-2.3 4.5-4.5 4.7.4.3.7.9.7 1.8v2.7c0 .3.2.6.7.5A9.6 9.6 0 0 0 12 2.4Z"
        fill="currentColor"
        fillRule="evenodd"
      />
    </svg>
  )
}

interface GitProviderGuide {
  type: GitProvider['type']
  createUrl?: string
  appName: string
  homepageUrl: string
  callbackUrl: string
  scopes: string
  docsUrl: string
}

function OAuthAppGuide({ guide }: { guide: GitProviderGuide }) {
  const { t } = useTranslation()
  return (
    <Alert>
      <Info />
      <AlertTitle>{t('codeRepositoriesView.oauthAppGuideTitle')}</AlertTitle>
      <AlertDescription className="gap-3">
        <p>{t(`codeRepositoriesView.${guide.type}OAuthGuide`)}</p>
        <div className="grid w-full gap-2 rounded-md bg-muted/70 p-3 text-xs text-foreground">
          <GuideValue label={t('codeRepositoriesView.oauthAppName')} value={guide.appName} />
          <GuideValue label={t('codeRepositoriesView.oauthHomepageUrl')} value={guide.homepageUrl} />
          <GuideValue important label={t('codeRepositoriesView.oauthCallbackUrl')} value={guide.callbackUrl} />
          <GuideValue label={t('codeRepositoriesView.oauthScopes')} value={guide.scopes} />
        </div>
        <div className="flex flex-wrap gap-2">
          {guide.createUrl && (
            <Button type="button" variant="secondary" onClick={() => window.open(guide.createUrl, '_blank', 'noopener,noreferrer')}>
              <ExternalLink size={16} />
              {t('codeRepositoriesView.openOAuthAppCreatePage')}
            </Button>
          )}
          <Button type="button" variant="secondary" onClick={() => window.open(guide.docsUrl, '_blank', 'noopener,noreferrer')}>
            <ExternalLink size={16} />
            {t('codeRepositoriesView.openOfficialDocs')}
          </Button>
        </div>
      </AlertDescription>
    </Alert>
  )
}

function GuideValue({ important, label, value }: { important?: boolean, label: string, value: string }) {
  const { t } = useTranslation()
  return (
    <div className="grid gap-1 sm:grid-cols-[9rem_1fr_auto] sm:items-center">
      <span className="text-muted-foreground">{label}</span>
      <code className={important ? 'break-all font-mono text-primary' : 'break-all font-mono'}>{value}</code>
      <Button
        className="w-fit"
        type="button"
        variant="ghost"
        onClick={() => {
          navigator.clipboard.writeText(value)
          toast.success(t('codeRepositoriesView.copied'))
        }}
      >
        <Copy size={14} />
        {t('common.copy')}
      </Button>
    </div>
  )
}

function CredentialOAuthGuide({ provider }: { provider: GitProvider }) {
  const { t } = useTranslation()
  const { debugOverride } = useSession()
  if (debugOverride) {
    return (
      <Alert>
        <Info />
        <AlertTitle>{t('codeRepositoriesView.oauthDebugBlockedTitle')}</AlertTitle>
        <AlertDescription>{t('codeRepositoriesView.oauthDebugBlockedDescription')}</AlertDescription>
      </Alert>
    )
  }
  if (isGitOAuthReady(provider)) {
    return (
      <Alert>
        <Info />
        <AlertTitle>{t('codeRepositoriesView.oauthReadyTitle')}</AlertTitle>
        <AlertDescription>
          <p>{t('codeRepositoriesView.oauthReadyDescription', { provider: provider.name })}</p>
          <Button className="mt-2" type="button" variant="secondary" onClick={() => { window.location.href = gitOAuthStartUrl(provider.id, '/code-repositories', window.location.origin) }}>
            <LinkIcon size={16} />
            {t('codeRepositoriesView.oauthConnect', { provider: provider.name })}
          </Button>
        </AlertDescription>
      </Alert>
    )
  }
  return (
    <Alert>
      <Info />
      <AlertTitle>{t('codeRepositoriesView.oauthManualOnlyTitle')}</AlertTitle>
      <AlertDescription>
        {provider.authType === 'oauth'
          ? t('codeRepositoriesView.oauthManualOnlyDescription', { provider: provider.name })
          : t('codeRepositoriesView.patManualOnlyDescription', { provider: provider.name })}
      </AlertDescription>
    </Alert>
  )
}

function splitText(value?: string) {
  return (value ?? '').split(',').map(item => item.trim()).filter(Boolean)
}

function projectScopeBadges(projectIds: string[] | undefined, projectMap: Record<string, string>) {
  return (projectIds ?? []).map(projectId => (
    <StatusBadge key={projectId}>{projectMap[projectId] ?? projectId}</StatusBadge>
  ))
}

function isGitOAuthReady(provider: GitProvider) {
  return provider.enabled && provider.authType === 'oauth' && provider.clientId.trim() !== '' && provider.clientSecretSet
}

function gitProviderGuide(type: GitProvider['type'], baseUrl?: string, name?: string): GitProviderGuide {
  const normalizedBaseUrl = normalizeGitBaseUrl(type, baseUrl)
  const callbackUrl = `${apiBaseOrigin()}/api/v1/git/oauth/callback`
  const appName = name?.trim() || (type === 'github' ? 'Liteyuki DevOps' : 'Liteyuki DevOps')
  if (type === 'gitea') {
    return {
      appName,
      callbackUrl,
      createUrl: normalizedBaseUrl ? `${normalizedBaseUrl}/user/settings/applications` : undefined,
      docsUrl: 'https://docs.gitea.com/development/oauth2-provider',
      homepageUrl: apiBaseOrigin(),
      scopes: 'read:repository, write:repository, read:user',
      type,
    }
  }
  if (type === 'gitlab') {
    return {
      appName,
      callbackUrl,
      docsUrl: 'https://docs.gitlab.com/integration/oauth_provider/',
      homepageUrl: apiBaseOrigin(),
      scopes: 'read_user, read_repository, write_repository',
      type,
    }
  }
  return {
    appName,
    callbackUrl,
    createUrl: 'https://github.com/settings/applications/new',
    docsUrl: 'https://docs.github.com/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app',
    homepageUrl: apiBaseOrigin(),
    scopes: 'repo, read:user',
    type: 'github',
  }
}

function normalizeGitBaseUrl(type: GitProvider['type'], baseUrl?: string) {
  const trimmed = baseUrl?.trim().replace(/\/+$/, '')
  if (trimmed)
    return trimmed
  return type === 'github' ? 'https://github.com' : ''
}

function gitProviderFaviconUrl(type: GitProvider['type'], baseUrl?: string) {
  const normalized = normalizeGitBaseUrl(type, baseUrl)
  if (!normalized)
    return ''
  try {
    return `${new URL(normalized).origin}/favicon.ico`
  }
  catch {
    return ''
  }
}
