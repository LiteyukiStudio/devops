import type { ArtifactRegistry, RegistryCredential, RegistryRepositoryItem } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Container, KeyRound, Plus, RefreshCw, Search, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'
import { api } from '@/api/client'
import { CheckboxField } from '@/components/common/checkbox-field'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ContentTabs } from '@/components/common/content-tabs'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { ProjectSpaceMultiSelect } from '@/components/common/project-space-select'
import { StatusBadge, StatusValueBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { TabsContent } from '@/components/ui/tabs'
import i18next from '@/i18n'

const registrySchema = z.object({
  name: z.string().min(1, i18next.t('registriesPage.registryNameRequired')),
  provider: z.enum(['harbor', 'dockerhub', 'gitea-registry']),
  endpoint: z.string().url(i18next.t('registriesPage.validUrlRequired')),
  scope: z.enum(['global', 'project', 'user']),
  ownerRef: z.string(),
  projectIds: z.array(z.string()),
  isDefault: z.boolean(),
  capabilitiesText: z.string(),
}).superRefine((values, ctx) => {
  if (values.scope === 'project' && values.projectIds.length === 0) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['projectIds'],
      message: i18next.t('projectSpaces.selectProject'),
    })
  }
})

const credentialSchema = z.object({
  registryId: z.string().min(1, i18next.t('registriesPage.registryRequired')),
  name: z.string().min(1, i18next.t('registriesPage.credentialNameRequired')),
  username: z.string(),
  password: z.string(),
  token: z.string(),
  scope: z.enum(['push-pull', 'push', 'pull']),
  accessScope: z.enum(['personal', 'registry']),
}).refine(values => values.password.trim() !== '' || values.token.trim() !== '', {
  message: i18next.t('registriesPage.passwordOrTokenRequired'),
  path: ['password'],
})

const imageSchema = z.object({
  projectId: z.string(),
  applicationId: z.string(),
  registryId: z.string().min(1, i18next.t('registriesPage.registryRequired')),
  repository: z.string().min(1, i18next.t('registriesPage.repositoryRequired')),
  tag: z.string(),
  digest: z.string(),
  sourceCommit: z.string(),
  buildRunId: z.string(),
  sourceType: z.enum(['manual-image', 'build']),
  scanStatus: z.enum(['unknown', 'pending', 'scanning', 'passed', 'failed']),
})

type RegistryForm = z.infer<typeof registrySchema>
type CredentialForm = z.infer<typeof credentialSchema>
type ImageForm = z.infer<typeof imageSchema>
type CredentialWithRegistry = RegistryCredential & { registryName: string }

const IMAGE_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const registryDefaults: RegistryForm = {
  name: '',
  provider: 'harbor',
  endpoint: '',
  scope: 'global',
  ownerRef: '',
  projectIds: [],
  isDefault: false,
  capabilitiesText: 'push,pull,tags,digest',
}

export function RegistriesPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editingRegistry, setEditingRegistry] = useState<ArtifactRegistry | null>(null)
  const [registryToDelete, setRegistryToDelete] = useState<ArtifactRegistry | null>(null)
  const [credentialToDelete, setCredentialToDelete] = useState<RegistryCredential | null>(null)
  const [selectedRegistryId, setSelectedRegistryId] = useState('')
  const [activeTab, setActiveTab] = useState('registries')
  const [registryDialogOpen, setRegistryDialogOpen] = useState(false)
  const [credentialDialogOpen, setCredentialDialogOpen] = useState(false)
  const [imageDialogOpen, setImageDialogOpen] = useState(false)
  const [registryPage, setRegistryPage] = useState(1)
  const [registryPageSize, setRegistryPageSize] = useState(10)
  const [credentialPage, setCredentialPage] = useState(1)
  const [credentialPageSize, setCredentialPageSize] = useState(10)
  const [imagePage, setImagePage] = useState(1)
  const [imagePageSize, setImagePageSize] = useState(20)
  const [imageRepositorySearch, setImageRepositorySearch] = useState('')
  const [imageRepositoryResultsOpen, setImageRepositoryResultsOpen] = useState(false)
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects })
  const registries = useQuery({
    queryKey: ['registries', registryPage, registryPageSize],
    queryFn: () => api.listRegistriesPage({ page: registryPage, pageSize: registryPageSize, sortBy: 'createdAt', sortOrder: 'desc' }),
  })
  const registryItems = registries.data?.items ?? []
  const registryOptions = useQuery({ queryKey: ['registries', 'options'], queryFn: () => api.listRegistries() })
  const images = useQuery({
    queryKey: ['container-images', imagePage, imagePageSize],
    queryFn: () => api.listContainerImages({ page: imagePage, pageSize: imagePageSize, sortBy: 'createdAt', sortOrder: 'desc' }),
  })
  const projectMap = useMemo(() => Object.fromEntries((projects.data ?? []).map(project => [project.id, project])), [projects.data])
  const allCredentials = useQuery({
    queryKey: ['registry-credentials', 'all', registryItems.map(registry => registry.id).join(',')],
    queryFn: async () => {
      const results = await Promise.all(registryItems.map(async (registry) => {
        try {
          const items = await api.listRegistryCredentials(registry.id)
          return items.map(credential => ({ ...credential, registryName: registry.name }))
        }
        catch {
          return [] as CredentialWithRegistry[]
        }
      }))
      return results.flat().sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime())
    },
    enabled: registries.isSuccess,
  })
  const credentials = useQuery({
    queryKey: ['registry-credentials', selectedRegistryId, credentialPage, credentialPageSize],
    queryFn: () => api.listRegistryCredentialsPage(selectedRegistryId, { page: credentialPage, pageSize: credentialPageSize, sortBy: 'createdAt', sortOrder: 'desc' }),
    enabled: Boolean(selectedRegistryId),
  })

  const registryForm = useForm<RegistryForm>({
    resolver: zodResolver(registrySchema),
    mode: 'onChange',
    defaultValues: registryDefaults,
  })
  const credentialForm = useForm<CredentialForm>({
    resolver: zodResolver(credentialSchema),
    mode: 'onChange',
    defaultValues: { accessScope: 'personal', registryId: '', name: 'default', username: '', password: '', token: '', scope: 'push-pull' },
  })
  const imageForm = useForm<ImageForm>({
    resolver: zodResolver(imageSchema),
    mode: 'onChange',
    defaultValues: {
      projectId: '',
      applicationId: '',
      registryId: '',
      repository: '',
      tag: 'latest',
      digest: '',
      sourceCommit: '',
      buildRunId: '',
      sourceType: 'manual-image',
      scanStatus: 'unknown',
    },
  })
  const imageRegistryId = imageForm.watch('registryId')
  const imageRepository = imageForm.watch('repository')
  const imageRepositoryResults = useQuery({
    queryKey: ['registry-repositories', imageRegistryId, imageRepositorySearch],
    queryFn: () => api.searchRegistryRepositories(imageRegistryId, { search: imageRepositorySearch, page: 1, pageSize: 10 }),
    enabled: Boolean(imageRegistryId && imageRepositorySearch.trim().length >= 2),
  })
  const imageTags = useQuery({
    queryKey: ['registry-tags', imageRegistryId, imageRepository],
    queryFn: () => api.listRegistryRepositoryTags(imageRegistryId, imageRepository, 20),
    enabled: Boolean(imageRegistryId && imageRepository.trim()),
  })

  const saveRegistry = useMutation({
    mutationFn: (values: RegistryForm) => {
      const payload = {
        name: values.name,
        provider: values.provider,
        endpoint: values.endpoint,
        scope: values.scope,
        ownerRef: '',
        projectIds: values.scope === 'project' ? values.projectIds : [],
        isDefault: values.isDefault,
        capabilities: splitText(values.capabilitiesText),
      }
      if (editingRegistry)
        return api.updateRegistry(editingRegistry.id, payload)
      return api.createRegistry(payload)
    },
    onSuccess: () => {
      toast.success(editingRegistry ? t('registriesPage.registryUpdated') : t('registriesPage.registryCreated'))
      setRegistryDialogOpen(false)
      setEditingRegistry(null)
      registryForm.reset(registryDefaults)
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
    onError: error => toast.error(error.message),
  })

  const createCredential = useMutation({
    mutationFn: (values: CredentialForm) => {
      const registry = (registryOptions.data ?? []).find(item => item.id === values.registryId)
      return api.createRegistryCredential(values.registryId, {
        ...values,
        accessScope: registry?.scope === 'global' ? 'personal' : values.accessScope,
      })
    },
    onSuccess: (_, values) => {
      toast.success(t('registriesPage.credentialSaved'))
      setCredentialDialogOpen(false)
      setSelectedRegistryId(values.registryId)
      credentialForm.reset({ accessScope: 'personal', registryId: values.registryId, name: 'default', username: '', password: '', token: '', scope: 'push-pull' })
      queryClient.invalidateQueries({ queryKey: ['registry-credentials', values.registryId] })
      queryClient.invalidateQueries({ queryKey: ['registry-credentials', 'all'] })
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteRegistry = useMutation({
    mutationFn: (registryId: string) => api.deleteRegistry(registryId),
    onSuccess: () => {
      toast.success(t('registriesPage.registryDeleted'))
      setRegistryToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteCredential = useMutation({
    mutationFn: ({ registryId, credentialId }: { registryId: string, credentialId: string }) => api.deleteRegistryCredential(registryId, credentialId),
    onSuccess: (_, values) => {
      toast.success(t('registriesPage.credentialDeleted'))
      setCredentialToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['registry-credentials', values.registryId] })
      queryClient.invalidateQueries({ queryKey: ['registry-credentials', 'all'] })
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
    onError: error => toast.error(error.message),
  })

  const testRegistry = useMutation({
    mutationFn: api.testRegistry,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(result.message)
        return
      }
      toast.error(result.message)
    },
    onError: error => toast.error(error.message),
  })

  const createImage = useMutation({
    mutationFn: api.createContainerImage,
    onSuccess: () => {
      toast.success(t('registriesPage.imageCreated'))
      setImageDialogOpen(false)
      imageForm.reset()
      queryClient.invalidateQueries({ queryKey: ['container-images'] })
    },
    onError: error => toast.error(error.message),
  })
  const beginEdit = (registry: ArtifactRegistry) => {
    setEditingRegistry(registry)
    registryForm.reset({
      name: registry.name,
      provider: registry.provider,
      endpoint: registry.endpoint,
      scope: registry.scope,
      ownerRef: registry.ownerRef,
      projectIds: registry.projectIds ?? [],
      isDefault: registry.isDefault,
      capabilitiesText: registry.capabilities.join(', '),
    })
    setRegistryDialogOpen(true)
  }

  const selectedRegistry = (registryOptions.data ?? registryItems).find(registry => registry.id === selectedRegistryId)
  const credentialRegistry = (registryOptions.data ?? registryItems).find(registry => registry.id === credentialForm.watch('registryId'))
  const credentialRegistryIsGlobal = credentialRegistry?.scope === 'global'
  const visibleCredentials: CredentialWithRegistry[] = selectedRegistryId
    ? (credentials.data?.items ?? []).map(credential => ({ ...credential, registryName: selectedRegistry?.name ?? '' }))
    : (allCredentials.data ?? [])

  const selectImageRepository = (repository: RegistryRepositoryItem) => {
    imageForm.setValue('repository', repository.name, { shouldDirty: true, shouldValidate: true })
    imageForm.setValue('tag', '', { shouldDirty: true, shouldValidate: true })
    setImageRepositorySearch(repository.name)
    setImageRepositoryResultsOpen(false)
  }

  return (
    <div className="grid gap-6">
      <ContentTabs
        tabs={[
          { value: 'registries', label: t('registriesPage.registriesTab') },
          { value: 'credentials', label: t('registriesPage.credentialsTab') },
          { value: 'images', label: t('registriesPage.imagesTab') },
        ]}
        tools={(
          <>
            {activeTab === 'registries' && (
              <Button
                onClick={() => {
                  setEditingRegistry(null)
                  registryForm.reset(registryDefaults)
                  setRegistryDialogOpen(true)
                }}
              >
                <Plus size={16} />
                {t('registriesPage.createRegistry')}
              </Button>
            )}
            {activeTab === 'credentials' && (
              <div className="flex flex-nowrap items-center justify-end gap-2">
                <Select className="h-9" containerClassName="w-40 shrink-0" value={selectedRegistryId} aria-label={t('registriesPage.selectRegistryTitle')} onChange={event => setSelectedRegistryId(event.target.value)}>
                  <option value="">{t('registriesPage.allRegistries')}</option>
                  {(registryOptions.data ?? registryItems).map(registry => (
                    <option key={registry.id} value={registry.id}>{registry.name}</option>
                  ))}
                </Select>
                <Button
                  className="shrink-0 whitespace-nowrap"
                  onClick={() => {
                    credentialForm.setValue('registryId', selectedRegistryId, { shouldValidate: true })
                    credentialForm.setValue('accessScope', 'personal', { shouldValidate: true })
                    setCredentialDialogOpen(true)
                  }}
                >
                  <KeyRound size={16} />
                  {t('registriesPage.createCredentialTitle')}
                </Button>
              </div>
            )}
            {activeTab === 'images' && (
              <Button
                onClick={() => {
                  imageForm.setValue('registryId', selectedRegistryId, { shouldValidate: true })
                  setImageDialogOpen(true)
                }}
              >
                <Container size={16} />
                {t('registriesPage.recordImage')}
              </Button>
            )}
          </>
        )}
        value={activeTab}
        onValueChange={setActiveTab}
      >

        <TabsContent value="registries">
          {registries.isError && <ErrorState title={t('registriesPage.loadFailedTitle')} description={t('registriesPage.loadFailedDescription')} />}
          <DataList
            columns={[
              {
                key: 'name',
                header: t('registriesPage.name'),
                render: registry => (
                  <button
                    className="grid min-w-0 text-left"
                    type="button"
                    onClick={() => {
                      setSelectedRegistryId(registry.id)
                      setCredentialPage(1)
                      setActiveTab('credentials')
                    }}
                  >
                    <span className="truncate font-medium">{registry.name}</span>
                    <span className="truncate text-sm text-muted-foreground">{registry.endpoint}</span>
                  </button>
                ),
              },
              { key: 'provider', header: t('registriesPage.provider'), render: registry => <StatusBadge>{registry.provider}</StatusBadge> },
              {
                key: 'scope',
                header: t('common.scope'),
                render: registry => (
                  <div className="flex flex-wrap gap-2">
                    <StatusBadge>{registry.scope}</StatusBadge>
                    {projectScopeBadges(registry.projectIds, projectMap)}
                    {registry.isDefault && <StatusBadge>{t('common.default')}</StatusBadge>}
                  </div>
                ),
              },
              { key: 'capabilities', header: t('registriesPage.capabilities'), render: registry => <span className="text-sm text-muted-foreground">{registry.capabilities.join(', ') || t('registriesPage.noCapabilities')}</span> },
              {
                key: 'actions',
                header: t('common.actions'),
                className: 'text-right whitespace-nowrap',
                render: registry => (
                  <div className="flex justify-end gap-2">
                    <EditActionButton type="button" label={t('edit')} onClick={() => beginEdit(registry)} />
                    <Button disabled={testRegistry.isPending} type="button" variant="ghost" onClick={() => testRegistry.mutate(registry.id)}>
                      <RefreshCw size={16} />
                      {t('registriesPage.test')}
                    </Button>
                    <Button aria-label={t('registriesPage.deleteRegistryAria')} type="button" variant="ghost" onClick={() => setRegistryToDelete(registry)}>
                      <Trash2 size={16} />
                    </Button>
                  </div>
                ),
              },
            ]}
            emptyTitle={t('registriesPage.emptyTitle')}
            emptyDescription={t('registriesPage.emptyDescription')}
            items={registryItems}
            pagination={{
              page: registries.data?.page ?? registryPage,
              pageSize: registries.data?.pageSize ?? registryPageSize,
              pageSizeOptions: PAGE_SIZE_OPTIONS,
              total: registries.data?.total ?? 0,
              totalPages: registries.data?.totalPages ?? 0,
              pageInfoLabel: t('pagination.pageInfo', {
                page: registries.data?.page ?? registryPage,
                totalPages: registries.data?.totalPages ?? 0,
                total: registries.data?.total ?? 0,
              }),
              onPageChange: setRegistryPage,
              onPageSizeChange: (nextPageSize) => {
                setRegistryPageSize(nextPageSize)
                setRegistryPage(1)
              },
            }}
            rowKey={registry => registry.id}
          />
        </TabsContent>

        <TabsContent value="credentials">
          <DataList
            columns={[
              {
                key: 'name',
                header: t('registriesPage.name'),
                render: credential => (
                  <div className="min-w-0">
                    <div className="truncate font-medium">{credential.name}</div>
                    <p className="truncate text-sm text-muted-foreground">{credential.username || t('registriesPage.tokenOnly')}</p>
                  </div>
                ),
              },
              { key: 'registry', header: t('registries'), render: credential => credential.registryName },
              { key: 'usage', header: t('registriesPage.usage'), render: credential => <StatusBadge>{credential.scope}</StatusBadge> },
              { key: 'access', header: t('registriesPage.credentialAccessScope'), render: credential => <StatusBadge>{credential.accessScope === 'registry' ? t('registriesPage.credentialAccessScopeRegistry') : t('registriesPage.credentialAccessScopePersonal')}</StatusBadge> },
              {
                key: 'secret',
                header: t('registriesPage.credential'),
                render: credential => (
                  <div className="flex flex-wrap gap-2">
                    {credential.passwordSet && <StatusBadge>{t('registriesPage.passwordSet')}</StatusBadge>}
                    {credential.tokenSet && <StatusBadge>{t('registriesPage.tokenSet')}</StatusBadge>}
                  </div>
                ),
              },
              {
                key: 'actions',
                header: t('common.actions'),
                className: 'text-right whitespace-nowrap',
                render: credential => (
                  <Button aria-label={t('registriesPage.deleteCredentialAria')} variant="ghost" onClick={() => setCredentialToDelete(credential)}>
                    <Trash2 size={16} />
                  </Button>
                ),
              },
            ]}
            emptyTitle={t('registriesPage.noCredentialsTitle')}
            emptyDescription={t('registriesPage.noCredentialsDescription')}
            items={visibleCredentials}
            pagination={selectedRegistryId
              ? {
                  page: credentials.data?.page ?? credentialPage,
                  pageSize: credentials.data?.pageSize ?? credentialPageSize,
                  pageSizeOptions: PAGE_SIZE_OPTIONS,
                  total: credentials.data?.total ?? 0,
                  totalPages: credentials.data?.totalPages ?? 0,
                  pageInfoLabel: t('pagination.pageInfo', {
                    page: credentials.data?.page ?? credentialPage,
                    totalPages: credentials.data?.totalPages ?? 0,
                    total: credentials.data?.total ?? 0,
                  }),
                  onPageChange: setCredentialPage,
                  onPageSizeChange: (nextPageSize) => {
                    setCredentialPageSize(nextPageSize)
                    setCredentialPage(1)
                  },
                }
              : undefined}
            rowKey={credential => credential.id}
          />
        </TabsContent>

        <TabsContent value="images">
          <DataList
            columns={[
              { key: 'image', header: t('registriesPage.image'), render: image => <code className="block max-w-xl truncate rounded bg-background px-2 py-1 text-xs" title={image.imageRef}>{image.imageRef}</code> },
              { key: 'registry', header: t('registries'), render: image => (registryOptions.data ?? registryItems).find(registry => registry.id === image.registryId)?.name ?? image.registryId },
              { key: 'source', header: t('common.type'), render: image => <StatusBadge>{image.sourceType}</StatusBadge> },
              { key: 'scan', header: t('common.status'), render: image => <StatusValueBadge value={image.scanStatus} /> },
              { key: 'digest', header: t('registriesPage.digest'), render: image => image.digest ? <CheckCircle2 className="text-primary" size={16} /> : '-' },
            ]}
            emptyTitle={t('registriesPage.noImagesTitle')}
            emptyDescription={t('registriesPage.noImagesDescription')}
            items={images.data?.items ?? []}
            pagination={{
              page: images.data?.page ?? imagePage,
              pageSize: images.data?.pageSize ?? imagePageSize,
              pageSizeOptions: IMAGE_PAGE_SIZE_OPTIONS,
              total: images.data?.total ?? 0,
              totalPages: images.data?.totalPages ?? 0,
              pageInfoLabel: t('pagination.pageInfo', {
                page: images.data?.page ?? imagePage,
                totalPages: images.data?.totalPages ?? 0,
                total: images.data?.total ?? 0,
              }),
              onPageChange: setImagePage,
              onPageSizeChange: (nextPageSize) => {
                setImagePageSize(nextPageSize)
                setImagePage(1)
              },
            }}
            rowKey={image => image.id}
          />
        </TabsContent>

      </ContentTabs>

      <Dialog
        open={registryDialogOpen}
        onOpenChange={(open) => {
          setRegistryDialogOpen(open)
          if (!open) {
            setEditingRegistry(null)
            registryForm.reset(registryDefaults)
          }
        }}
      >
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editingRegistry ? t('registriesPage.editRegistryTitle') : t('registriesPage.createRegistryTitle')}</DialogTitle>
            <DialogDescription>{t('registriesPage.description')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={registryForm.handleSubmit(values => saveRegistry.mutate(values))}>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={registryForm.formState.errors.name?.message} hint={t('registriesPage.registryNameHint')} label={t('registriesPage.name')} required>
                <Input {...registryForm.register('name')} aria-invalid={Boolean(registryForm.formState.errors.name)} placeholder={t('registriesPage.registryNamePlaceholder')} />
              </Field>
              <Field error={registryForm.formState.errors.provider?.message} hint={t('registriesPage.providerHint')} label={t('registriesPage.provider')} required>
                <Select {...registryForm.register('provider')} aria-invalid={Boolean(registryForm.formState.errors.provider)}>
                  <option value="harbor">{t('registriesPage.providerHarbor')}</option>
                  <option value="dockerhub">{t('registriesPage.providerDockerHub')}</option>
                  <option value="gitea-registry">{t('registriesPage.providerGiteaRegistry')}</option>
                </Select>
              </Field>
            </div>
            <Field error={registryForm.formState.errors.endpoint?.message} hint={t('registriesPage.endpointHint')} label={t('registriesPage.endpoint')} required>
              <Input {...registryForm.register('endpoint')} aria-invalid={Boolean(registryForm.formState.errors.endpoint)} placeholder={t('registriesPage.endpointPlaceholder')} />
            </Field>
            <Field error={registryForm.formState.errors.scope?.message} hint={t('registriesPage.registryScopeHint')} label={t('registriesPage.scope')} required>
              <Select {...registryForm.register('scope')} aria-invalid={Boolean(registryForm.formState.errors.scope)}>
                <option value="global">{t('registriesPage.scopeGlobal')}</option>
                <option value="project">{t('registriesPage.scopeProject')}</option>
                <option value="user">{t('registriesPage.scopeUser')}</option>
              </Select>
            </Field>
            {registryForm.watch('scope') === 'project' && (
              <Field error={registryForm.formState.errors.projectIds?.message} hint={t('registriesPage.ownerProjectHint')} label={t('registriesPage.ownerProject')}>
                <ProjectSpaceMultiSelect
                  projects={projects.data ?? []}
                  value={registryForm.watch('projectIds')}
                  onChange={value => registryForm.setValue('projectIds', value, { shouldDirty: true, shouldValidate: true })}
                />
              </Field>
            )}
            <Field error={registryForm.formState.errors.capabilitiesText?.message} hint={t('registriesPage.capabilitiesHint')} label={t('registriesPage.capabilities')}>
              <Input {...registryForm.register('capabilitiesText')} aria-invalid={Boolean(registryForm.formState.errors.capabilitiesText)} />
            </Field>
            <CheckboxField {...registryForm.register('isDefault')}>
              {t('registriesPage.setAsDefault')}
            </CheckboxField>
            <DialogFooter>
              <Button disabled={saveRegistry.isPending || !registryForm.formState.isValid} type="submit">
                <Plus size={16} />
                {editingRegistry ? t('registriesPage.saveRegistry') : t('registriesPage.createRegistry')}
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
            credentialForm.reset({ accessScope: 'personal', registryId: selectedRegistryId, name: 'default', username: '', password: '', token: '', scope: 'push-pull' })
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('registriesPage.createCredentialTitle')}</DialogTitle>
            <DialogDescription>{t('registriesPage.credentialRegistryHint')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={credentialForm.handleSubmit(values => createCredential.mutate(values))}>
            <Field error={credentialForm.formState.errors.registryId?.message} hint={t('registriesPage.credentialRegistryHint')} label={t('registries')} required>
              <Select
                {...credentialForm.register('registryId')}
                aria-invalid={Boolean(credentialForm.formState.errors.registryId)}
                onChange={(event) => {
                  credentialForm.setValue('registryId', event.target.value, { shouldValidate: true })
                  const registry = (registryOptions.data ?? registryItems).find(item => item.id === event.target.value)
                  if (registry?.scope === 'global')
                    credentialForm.setValue('accessScope', 'personal', { shouldValidate: true })
                  setSelectedRegistryId(event.target.value)
                }}
              >
                <option value="">{t('registriesPage.selectRegistry')}</option>
                {(registryOptions.data ?? registryItems).map(registry => (
                  <option key={registry.id} value={registry.id}>{registry.name}</option>
                ))}
              </Select>
            </Field>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={credentialForm.formState.errors.name?.message} hint={t('registriesPage.credentialNameHint')} label={t('registriesPage.name')} required>
                <Input {...credentialForm.register('name')} aria-invalid={Boolean(credentialForm.formState.errors.name)} />
              </Field>
              <Field error={credentialForm.formState.errors.scope?.message} hint={t('registriesPage.credentialScopeHint')} label={t('registriesPage.usage')} required>
                <Select {...credentialForm.register('scope')} aria-invalid={Boolean(credentialForm.formState.errors.scope)}>
                  <option value="push-pull">{t('registriesPage.credentialScopePushPull')}</option>
                  <option value="push">{t('registriesPage.credentialScopePush')}</option>
                  <option value="pull">{t('registriesPage.credentialScopePull')}</option>
                </Select>
              </Field>
            </div>
            <Field error={credentialForm.formState.errors.accessScope?.message} hint={credentialRegistryIsGlobal ? t('registriesPage.credentialAccessScopeGlobalHint') : t('registriesPage.credentialAccessScopeHint')} label={t('registriesPage.credentialAccessScope')} required>
              <Select {...credentialForm.register('accessScope')} aria-invalid={Boolean(credentialForm.formState.errors.accessScope)} disabled={credentialRegistryIsGlobal}>
                <option value="personal">{t('registriesPage.credentialAccessScopePersonal')}</option>
                {!credentialRegistryIsGlobal && <option value="registry">{t('registriesPage.credentialAccessScopeRegistry')}</option>}
              </Select>
            </Field>
            <Field error={credentialForm.formState.errors.username?.message} hint={t('registriesPage.usernameHint')} label={t('registriesPage.username')}>
              <Input {...credentialForm.register('username')} aria-invalid={Boolean(credentialForm.formState.errors.username)} />
            </Field>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={credentialForm.formState.errors.password?.message} hint={t('registriesPage.passwordHint')} label={t('registriesPage.password')}>
                <Input {...credentialForm.register('password')} aria-invalid={Boolean(credentialForm.formState.errors.password)} type="password" />
              </Field>
              <Field error={credentialForm.formState.errors.token?.message} hint={t('registriesPage.tokenHint')} label={t('registriesPage.token')}>
                <Input {...credentialForm.register('token')} aria-invalid={Boolean(credentialForm.formState.errors.token)} type="password" />
              </Field>
            </div>
            <DialogFooter>
              <Button disabled={createCredential.isPending || !credentialForm.formState.isValid} type="submit">
                <KeyRound size={16} />
                {t('registriesPage.saveCredential')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={imageDialogOpen}
        onOpenChange={(open) => {
          setImageDialogOpen(open)
          if (!open)
            imageForm.reset()
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('registriesPage.recordImageTitle')}</DialogTitle>
            <DialogDescription>{t('registriesPage.imageRegistryHint')}</DialogDescription>
          </DialogHeader>
          <form className="grid gap-3" onSubmit={imageForm.handleSubmit(values => createImage.mutate(values))}>
            <Field error={imageForm.formState.errors.registryId?.message} hint={t('registriesPage.imageRegistryHint')} label={t('registries')} required>
              <Select
                {...imageForm.register('registryId', {
                  onChange: () => {
                    imageForm.setValue('repository', '', { shouldDirty: true, shouldValidate: true })
                    imageForm.setValue('tag', '', { shouldDirty: true, shouldValidate: true })
                    setImageRepositorySearch('')
                    setImageRepositoryResultsOpen(false)
                  },
                })}
                aria-invalid={Boolean(imageForm.formState.errors.registryId)}
              >
                <option value="">{t('registriesPage.selectRegistry')}</option>
                {(registryOptions.data ?? registryItems).map(registry => (
                  <option key={registry.id} value={registry.id}>{registry.name}</option>
                ))}
              </Select>
            </Field>
            <Field error={imageForm.formState.errors.repository?.message} hint={t('registriesPage.repositoryHint')} label={t('registriesPage.repository')} required>
              <div className="flex gap-2">
                <Input
                  {...imageForm.register('repository')}
                  aria-invalid={Boolean(imageForm.formState.errors.repository)}
                  placeholder={t('registriesPage.repositoryPlaceholder')}
                  value={imageRepositorySearch}
                  onChange={(event) => {
                    setImageRepositorySearch(event.target.value)
                    imageForm.setValue('repository', event.target.value, { shouldDirty: true, shouldValidate: true })
                    setImageRepositoryResultsOpen(event.target.value.trim().length >= 2)
                  }}
                  onFocus={() => setImageRepositoryResultsOpen(Boolean(imageRegistryId && imageRepositorySearch.trim().length >= 2))}
                />
                <Button
                  disabled={!imageRegistryId || imageRepositorySearch.trim().length < 2 || imageRepositoryResults.isFetching}
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    setImageRepositoryResultsOpen(true)
                    imageRepositoryResults.refetch()
                  }}
                >
                  <Search size={16} />
                  {t('registriesPage.searchImages')}
                </Button>
              </div>
            </Field>
            {imageRegistryId && imageRepositoryResultsOpen && (
              <div className="grid max-h-52 gap-2 overflow-y-auto rounded-md border border-border p-2">
                {imageRepositoryResults.isFetching && (
                  <p className="px-3 py-2 text-sm text-muted-foreground">{t('registriesPage.searchingImages')}</p>
                )}
                {(imageRepositoryResults.data?.items ?? []).map(repository => (
                  <button
                    key={repository.name}
                    className="rounded-md px-3 py-2 text-left hover:bg-muted"
                    type="button"
                    onClick={() => selectImageRepository(repository)}
                  >
                    <span className="block text-sm font-medium">{repository.name}</span>
                    {repository.description && <span className="block truncate text-xs text-muted-foreground">{repository.description}</span>}
                  </button>
                ))}
                {imageRepositoryResults.isSuccess && imageRepositoryResults.data.items.length === 0 && (
                  <EmptyState description={t('registriesPage.noRepositoryResultsDescription')} title={t('registriesPage.noRepositoryResultsTitle')} variant="plain" />
                )}
                {imageRepositoryResults.isError && (
                  <EmptyState description={t('registriesPage.repositorySearchFailedDescription')} title={t('registriesPage.repositorySearchFailedTitle')} variant="plain" />
                )}
              </div>
            )}
            <div className="grid gap-3 sm:grid-cols-2">
              <Field error={imageForm.formState.errors.tag?.message} hint={t('registriesPage.tagHint')} label={t('registriesPage.tag')}>
                <Input
                  {...imageForm.register('tag')}
                  aria-invalid={Boolean(imageForm.formState.errors.tag)}
                  list="registry-image-tag-options"
                  placeholder={imageTags.isFetching ? t('registriesPage.loadingTags') : t('registriesPage.tagPlaceholder')}
                />
                <datalist id="registry-image-tag-options">
                  {(imageTags.data?.items ?? []).map(tag => (
                    <option key={tag.name} value={tag.name}>{tag.digest}</option>
                  ))}
                </datalist>
              </Field>
              <Field error={imageForm.formState.errors.digest?.message} hint={t('registriesPage.digestHint')} label={t('registriesPage.digest')}>
                <Input {...imageForm.register('digest')} aria-invalid={Boolean(imageForm.formState.errors.digest)} />
              </Field>
            </div>
            <DialogFooter>
              <Button disabled={createImage.isPending || !imageForm.formState.isValid} type="submit">
                <Container size={16} />
                {t('registriesPage.recordImage')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        confirmText={t('registriesPage.deleteRegistryConfirm')}
        description={t('registriesPage.deleteRegistryDescription', { name: registryToDelete?.name ?? '' })}
        open={Boolean(registryToDelete)}
        pending={deleteRegistry.isPending}
        title={t('registriesPage.deleteRegistryTitle')}
        onConfirm={() => registryToDelete && deleteRegistry.mutate(registryToDelete.id)}
        onOpenChange={open => !open && setRegistryToDelete(null)}
      />
      <ConfirmDialog
        confirmText={t('registriesPage.deleteCredentialConfirm')}
        description={t('registriesPage.deleteCredentialDescription', { name: credentialToDelete?.name ?? '' })}
        open={Boolean(credentialToDelete)}
        pending={deleteCredential.isPending}
        title={t('registriesPage.deleteCredentialTitle')}
        onConfirm={() => credentialToDelete && deleteCredential.mutate({ registryId: credentialToDelete.registryId, credentialId: credentialToDelete.id })}
        onOpenChange={open => !open && setCredentialToDelete(null)}
      />
    </div>
  )
}

function projectScopeBadges(projectIds: string[] | undefined, projectMap: Record<string, { name: string }>) {
  return (projectIds ?? []).map(projectId => (
    <StatusBadge key={projectId}>{projectMap[projectId]?.name ?? projectId}</StatusBadge>
  ))
}

function splitText(value: string) {
  return value.split(/[\n,]/).map(item => item.trim()).filter(Boolean)
}
