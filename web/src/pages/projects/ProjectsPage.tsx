import type { Project } from '@/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { FolderKanban, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'
import { api } from '@/api/client'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { PageHeader } from '@/components/common/page-header'
import { StatusBadge } from '@/components/common/status-badge'
import { formatSmartDateTime } from '@/components/common/time-format'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { PROJECT_SLUG_MAX_LENGTH } from '@/lib/slug-limits'

const schema = z.object({
  name: z.string().min(1, i18next.t('projectSpaces.nameRequired')),
  slug: z.string().min(1, i18next.t('projectSpaces.slugRequired')).max(PROJECT_SLUG_MAX_LENGTH, i18next.t('projectSpaces.slugMaxLength', { count: PROJECT_SLUG_MAX_LENGTH })).regex(/^[a-z0-9-]+$/, i18next.t('common.lowercaseSlugOnly')),
  description: z.string().optional(),
})

type ProjectForm = z.infer<typeof schema>

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const PROJECT_SORT_OPTIONS = ['lastUsed', 'useCount', 'createdAt', 'updatedAt', 'name'] as const
const PROJECT_SORT_ORDERS = ['desc', 'asc'] as const

type ProjectSortBy = typeof PROJECT_SORT_OPTIONS[number]
type ProjectSortOrder = typeof PROJECT_SORT_ORDERS[number]

export function ProjectsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editingProject, setEditingProject] = useState<Project | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [sortBy, setSortBy] = useState<ProjectSortBy>('lastUsed')
  const [sortOrder, setSortOrder] = useState<ProjectSortOrder>('desc')
  const projects = useQuery({
    queryKey: ['projects', 'page', page, pageSize, sortBy, sortOrder],
    queryFn: () => api.listProjectsPage({ page, pageSize, sortBy, sortOrder }),
  })
  const projectItems = Array.isArray(projects.data) ? projects.data : projects.data?.items ?? []
  const projectTotal = Array.isArray(projects.data) ? projects.data.length : projects.data?.total ?? 0
  const projectTotalPages = Array.isArray(projects.data) ? 1 : projects.data?.totalPages ?? 0
  const projectPage = Array.isArray(projects.data) ? 1 : projects.data?.page ?? page
  const projectPageSize = Array.isArray(projects.data) ? pageSize : projects.data?.pageSize ?? pageSize
  const form = useForm<ProjectForm>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { name: '', slug: '', description: '' },
  })

  const createProject = useMutation({
    mutationFn: api.createProject,
    onSuccess: () => {
      toast.success(t('projectSpaces.created'))
      form.reset()
      setDialogOpen(false)
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: error => toast.error(error.message),
  })

  const deleteProject = useMutation({
    mutationFn: api.deleteProject,
    onSuccess: () => {
      toast.success(t('projectSpaces.deleted'))
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: error => toast.error(error.message),
  })

  const updateProject = useMutation({
    mutationFn: ({ projectId, payload }: { projectId: string, payload: Pick<Project, 'slug' | 'name' | 'description'> }) =>
      api.updateProject(projectId, payload),
    onSuccess: () => {
      toast.success(t('projectSpaces.updated'))
      form.reset()
      setEditingProject(null)
      setDialogOpen(false)
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: error => toast.error(error.message),
  })

  return (
    <div className="grid gap-6">
      <PageHeader
        actions={(
          <div className="flex flex-wrap items-center justify-end gap-2">
            <Select
              aria-label={t('projectSpaces.sortBy')}
              containerClassName="w-40"
              value={sortBy}
              onChange={(event) => {
                setSortBy(event.target.value as ProjectSortBy)
                setPage(1)
              }}
            >
              {PROJECT_SORT_OPTIONS.map(option => (
                <option key={option} value={option}>{t(`projectSpaces.sort.${option}`)}</option>
              ))}
            </Select>
            <Select
              aria-label={t('projectSpaces.sortOrder')}
              containerClassName="w-32"
              value={sortOrder}
              onChange={(event) => {
                setSortOrder(event.target.value as ProjectSortOrder)
                setPage(1)
              }}
            >
              {PROJECT_SORT_ORDERS.map(order => (
                <option key={order} value={order}>{t(`projectSpaces.sortOrderOptions.${order}`)}</option>
              ))}
            </Select>
            <Button
              onClick={() => {
                setEditingProject(null)
                form.reset({ name: '', slug: '', description: '' })
                setDialogOpen(true)
              }}
            >
              <Plus size={16} />
              {t('projectSpaces.createTitle')}
            </Button>
          </div>
        )}
        description={t('projectSpaces.description')}
        title={t('projectSpaces.title')}
      />
      {projects.isError && <ErrorState title={t('projectSpaces.loadFailedTitle')} description={t('projectSpaces.loadFailedDescription')} />}
      <DataList
        columns={[
          {
            key: 'name',
            header: t('projectSpaces.title'),
            className: 'min-w-64 px-4 py-3 align-middle',
            render: project => <ProjectSummary project={project} />,
          },
          {
            key: 'slug',
            header: t('common.slug'),
            className: 'w-[18%] px-4 py-3 align-middle text-muted-foreground',
            render: project => <code className="rounded bg-background px-2 py-1 text-xs">{project.slug}</code>,
          },
          {
            key: 'namespaceStrategy',
            header: t('projectSpaces.namespaceStrategy'),
            className: 'w-[16%] px-4 py-3 align-middle',
            render: project => <StatusBadge>{project.namespaceStrategy}</StatusBadge>,
          },
          {
            key: 'usage',
            header: t('projectSpaces.usage'),
            className: 'w-[20%] px-4 py-3 align-middle',
            render: project => (
              <div className="grid gap-1">
                <span className="text-sm text-foreground">{project.lastUsedAt ? formatSmartDateTime(project.lastUsedAt, t) : t('projectSpaces.neverUsed')}</span>
                <span className="text-xs text-muted-foreground">{t('projectSpaces.useCount', { count: project.useCount ?? 0 })}</span>
              </div>
            ),
          },
          {
            key: 'actions',
            header: t('common.actions'),
            className: 'w-[1%] whitespace-nowrap px-4 py-3 align-middle text-right',
            render: project => (
              <div className="flex justify-end gap-2">
                <Link className="inline-flex h-9 items-center justify-center rounded-full border border-border bg-surface px-4 text-sm font-medium text-foreground transition hover:bg-muted" to={`/projects/${project.id}`}>
                  {t('projectSpaces.openWorkspace')}
                </Link>
                <EditActionButton
                  aria-label={t('projectSpaces.editAria')}
                  label={t('edit')}
                  onClick={() => {
                    setEditingProject(project)
                    form.reset({
                      name: project.name,
                      slug: project.slug,
                      description: project.description,
                    })
                    setDialogOpen(true)
                  }}
                />
                <ConfirmDialog
                  confirmText={t('projectSpaces.deleteConfirm')}
                  description={t('projectSpaces.deleteDescription', { name: project.name })}
                  pending={deleteProject.isPending}
                  title={t('projectSpaces.deleteTitle')}
                  onConfirm={() => deleteProject.mutate(project.id)}
                >
                  <Button aria-label={t('projectSpaces.deleteAria')} variant="ghost">
                    <Trash2 size={16} />
                  </Button>
                </ConfirmDialog>
              </div>
            ),
          },
        ]}
        emptyDescription={t('projectSpaces.emptyDescription')}
        emptyTitle={t('projectSpaces.emptyTitle')}
        items={projectItems}
        pagination={{
          page: projectPage,
          pageSize: projectPageSize,
          pageSizeOptions: PAGE_SIZE_OPTIONS,
          total: projectTotal,
          totalPages: projectTotalPages,
          pageInfoLabel: t('pagination.pageInfo', {
            page: projectPage,
            totalPages: projectTotalPages,
            total: projectTotal,
          }),
          onPageChange: setPage,
          onPageSizeChange: (nextPageSize) => {
            setPageSize(nextPageSize)
            setPage(1)
          },
        }}
        rowKey={project => project.id}
      />
      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) {
            setEditingProject(null)
            form.reset({ name: '', slug: '', description: '' })
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingProject ? t('projectSpaces.editTitle') : t('projectSpaces.createTitle')}</DialogTitle>
            <DialogDescription>{t('projectSpaces.description')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-3"
            onSubmit={form.handleSubmit((values) => {
              const payload = { ...values, description: values.description ?? '' }
              if (editingProject) {
                updateProject.mutate({ projectId: editingProject.id, payload })
                return
              }
              createProject.mutate(payload)
            })}
          >
            <Field error={form.formState.errors.name?.message} hint={t('projectSpaces.nameHint')} label={t('projectSpaces.name')} required>
              <Input {...form.register('name')} aria-invalid={Boolean(form.formState.errors.name)} placeholder={t('projectSpaces.namePlaceholder')} />
            </Field>
            <Field error={form.formState.errors.slug?.message} hint={t('projectSpaces.slugHint', { count: PROJECT_SLUG_MAX_LENGTH })} label={t('projectSpaces.slug')} required>
              <Input {...form.register('slug')} aria-invalid={Boolean(form.formState.errors.slug)} maxLength={PROJECT_SLUG_MAX_LENGTH} placeholder={t('projectSpaces.slugPlaceholder')} />
            </Field>
            <Field error={form.formState.errors.description?.message} hint={t('projectSpaces.descriptionHint')} label={t('projectSpaces.descriptionLabel')}>
              <Textarea {...form.register('description')} placeholder={t('projectSpaces.descriptionPlaceholder')} />
            </Field>
            <DialogFooter>
              <Button disabled={createProject.isPending || updateProject.isPending || !form.formState.isValid} type="submit">
                <Plus size={16} />
                {editingProject ? t('save') : t('create')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function ProjectSummary({ project }: { project: Project }) {
  const { t } = useTranslation()
  return (
    <div className="flex min-w-0 items-center gap-3">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <FolderKanban size={18} />
      </span>
      <div className="min-w-0">
        <Link className="truncate font-medium transition hover:text-primary" to={`/projects/${project.id}`}>
          {project.name}
        </Link>
        <p className="truncate text-sm text-muted-foreground">
          {project.description || t('common.noDescription')}
        </p>
      </div>
    </div>
  )
}
