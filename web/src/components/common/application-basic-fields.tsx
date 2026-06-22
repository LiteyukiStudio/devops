import type { UseFormRegisterReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ApplicationIconPicker } from '@/components/common/application-icon-picker'
import { FormField as Field } from '@/components/common/form-field'
import { Input } from '@/components/ui/input'
import { APPLICATION_SLUG_MAX_LENGTH } from '@/lib/slug-limits'

export function ApplicationBasicFields({
  compact = false,
  icon,
  nameError,
  nameField,
  onIconChange,
  slugError,
  slugField,
  slugMaxLength,
}: {
  compact?: boolean
  icon?: string
  nameError?: string
  nameField: UseFormRegisterReturn<'name'>
  onIconChange: (icon: string) => void
  slugError?: string
  slugField: UseFormRegisterReturn<'slug'>
  slugMaxLength?: number
}) {
  const { t } = useTranslation()
  const normalizedSlugMaxLength = slugMaxLength ?? APPLICATION_SLUG_MAX_LENGTH

  if (compact) {
    return (
      <div className="grid gap-3 md:grid-cols-[auto_minmax(0,1fr)_minmax(0,1fr)]">
        <Field hint={t('apps.iconHint')} label={t('apps.icon')}>
          <ApplicationIconPicker compact value={icon} onChange={onIconChange} />
        </Field>
        <Field error={nameError} hint={t('apps.nameHint')} label={t('apps.name')} required>
          <Input {...nameField} aria-invalid={Boolean(nameError)} placeholder={t('apps.namePlaceholder')} />
        </Field>
        <Field error={slugError} hint={t('apps.slugHint', { count: normalizedSlugMaxLength })} label={t('apps.slug')} required>
          <Input {...slugField} aria-invalid={Boolean(slugError)} maxLength={normalizedSlugMaxLength} placeholder={t('apps.slugPlaceholder')} />
        </Field>
      </div>
    )
  }

  return (
    <>
      <div className="grid gap-3 md:grid-cols-2">
        <Field error={nameError} hint={t('apps.nameHint')} label={t('apps.name')} required>
          <Input {...nameField} aria-invalid={Boolean(nameError)} placeholder={t('apps.namePlaceholder')} />
        </Field>
        <Field error={slugError} hint={t('apps.slugHint', { count: normalizedSlugMaxLength })} label={t('apps.slug')} required>
          <Input {...slugField} aria-invalid={Boolean(slugError)} maxLength={normalizedSlugMaxLength} placeholder={t('apps.slugPlaceholder')} />
        </Field>
      </div>
      <Field hint={t('apps.iconHint')} label={t('apps.icon')}>
        <ApplicationIconPicker value={icon} onChange={onIconChange} />
      </Field>
    </>
  )
}
