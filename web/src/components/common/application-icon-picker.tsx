import type { ApplicationIconName } from '@/components/common/application-icons'
import { useTranslation } from 'react-i18next'
import { APPLICATION_ICON_NAMES, applicationIconComponents, normalizeApplicationIconName } from '@/components/common/application-icons'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

export function ApplicationIcon({ className, name, size = 18 }: { className?: string, name?: string, size?: number }) {
  const Icon = applicationIconComponents[normalizeApplicationIconName(name)]
  return <Icon className={className} height={size} width={size} />
}

export function ApplicationIconPicker({ compact = false, disabled, value, onChange }: {
  compact?: boolean
  disabled?: boolean
  value?: string
  onChange: (value: ApplicationIconName) => void
}) {
  const { t } = useTranslation()
  const selected = normalizeApplicationIconName(value)

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          aria-label={t('apps.iconPickerAria')}
          className={compact ? 'size-9 rounded-md border-0 bg-transparent px-0 text-foreground shadow-none hover:bg-muted hover:text-foreground' : undefined}
          disabled={disabled}
          title={t(`apps.icons.${selected}`)}
          type="button"
          variant={compact ? 'ghost' : 'secondary'}
        >
          <ApplicationIcon name={selected} />
          {!compact && <span>{t(`apps.icons.${selected}`)}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-72 p-2">
        <div className="grid grid-cols-8 gap-1">
          {APPLICATION_ICON_NAMES.map((name) => {
            const active = name === selected
            return (
              <button
                key={name}
                aria-label={t(`apps.icons.${name}`)}
                className={cn(
                  'flex size-8 items-center justify-center rounded-md border border-transparent text-muted-foreground transition hover:border-border hover:bg-muted hover:text-foreground',
                  active && 'border-primary bg-primary/10 text-primary',
                )}
                title={t(`apps.icons.${name}`)}
                type="button"
                onClick={() => onChange(name)}
              >
                <ApplicationIcon name={name} />
              </button>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}
