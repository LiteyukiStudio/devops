import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FormField as Field } from '@/components/common/form-field'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'

interface BuildProviderFormValues {
  config: string
  enabled: boolean
  name: string
  ownerRef: string
  scope: 'global' | 'project' | 'user'
  slug: string
  type: 'platform'
}

export function BuildProviderFormFields({ form, projects }: {
  form: UseFormReturn<BuildProviderFormValues>
  projects: Array<{ id: string, name: string }>
}) {
  const { t } = useTranslation()
  const scope = form.watch('scope')

  return (
    <>
      <div className="grid gap-3 md:grid-cols-2">
        <Field error={form.formState.errors.slug?.message} hint={t('buildsPage.providerSlugHint')} label={t('buildsPage.providerSlug')} required>
          <Input {...form.register('slug', { required: true })} aria-invalid={Boolean(form.formState.errors.slug)} />
        </Field>
        <Field error={form.formState.errors.name?.message} hint={t('buildsPage.providerDisplayNameHint')} label={t('buildsPage.providerDisplayName')} required>
          <Input {...form.register('name', { required: true })} aria-invalid={Boolean(form.formState.errors.name)} />
        </Field>
      </div>
      <Field error={form.formState.errors.type?.message} label={t('common.type')} required>
        <Select {...form.register('type')} aria-invalid={Boolean(form.formState.errors.type)}>
          <option value="platform">{t('buildsPage.typePlatform')}</option>
        </Select>
      </Field>
      <Field error={form.formState.errors.scope?.message} label={t('common.scope')} required>
        <Select {...form.register('scope')} aria-invalid={Boolean(form.formState.errors.scope)}>
          <option value="global">{t('codeRepositoriesView.scopeGlobal')}</option>
          <option value="project">{t('codeRepositoriesView.scopeProject')}</option>
          <option value="user">{t('codeRepositoriesView.scopeUser')}</option>
        </Select>
      </Field>
      {scope === 'project' && (
        <Field error={form.formState.errors.ownerRef?.message} label={t('projectSpaces.title')} required>
          <Select {...form.register('ownerRef', { required: true })} aria-invalid={Boolean(form.formState.errors.ownerRef)}>
            <option value="">{t('common.select')}</option>
            {projects.map(project => <option key={project.id} value={project.id}>{project.name}</option>)}
          </Select>
        </Field>
      )}
      <Field error={form.formState.errors.config?.message} label={t('buildsPage.config')}>
        <Input {...form.register('config')} aria-invalid={Boolean(form.formState.errors.config)} />
      </Field>
      <label className="flex items-center gap-2 text-sm text-foreground">
        <input className="size-4 accent-primary" type="checkbox" {...form.register('enabled')} />
        {t('common.enabled')}
      </label>
    </>
  )
}
