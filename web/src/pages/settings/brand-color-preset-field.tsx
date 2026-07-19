import type { BrandColorPreset, UserBrandColorPreference } from '@/app/brand-theme'
import { Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { brandColorPresets, brandColorUsesDarkForeground, normalizeBrandColorPreset, normalizeUserBrandColorPreference } from '@/app/brand-theme'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

const brandColorPresetSet = new Set<string>(brandColorPresets)
const inheritValue = '__follow_platform__'

export function BrandColorPresetField({ ariaLabel, inheritedPreset, inheritLabel, value, options, onValueChange }: {
  ariaLabel: string
  inheritedPreset?: string
  inheritLabel?: string
  value: string
  options?: string[]
  onValueChange: (value: UserBrandColorPreference) => void
}) {
  const { t } = useTranslation()
  const availablePresets = (options ?? brandColorPresets).filter((preset): preset is BrandColorPreset => brandColorPresetSet.has(preset))
  const allowsInheritance = Boolean(inheritLabel)
  const selectedPreset = allowsInheritance ? normalizeUserBrandColorPreference(value) : normalizeBrandColorPreset(value)
  const platformPreset = normalizeBrandColorPreset(inheritedPreset)

  return (
    <RadioGroup
      aria-label={ariaLabel}
      className="flex flex-wrap items-center gap-2"
      value={selectedPreset || inheritValue}
      onValueChange={nextValue => onValueChange(nextValue === inheritValue ? '' : normalizeBrandColorPreset(nextValue))}
    >
      {allowsInheritance && (
        <BrandColorOption
          checked={selectedPreset === ''}
          id="brand-color-follow-platform"
          label={inheritLabel ?? ''}
          preset={platformPreset}
          value={inheritValue}
        />
      )}
      {availablePresets.map(preset => (
        <BrandColorOption
          key={preset}
          checked={selectedPreset === preset}
          id={`brand-color-${preset}`}
          label={t(`settings.brandColorPresets.${preset}`)}
          preset={preset}
          value={preset}
        />
      ))}
    </RadioGroup>
  )
}

function BrandColorOption({ checked, id, label, preset, value }: {
  checked: boolean
  id: string
  label: string
  preset: BrandColorPreset
  value: string
}) {
  const showLabel = value === inheritValue

  return (
    <div className="shrink-0">
      <RadioGroupItem id={id} className="peer sr-only" value={value} />
      <Tooltip>
        <TooltipTrigger asChild>
          <Label
            className={cn(
              'flex h-9 cursor-pointer items-center justify-center rounded-md border border-border bg-background transition-colors hover:border-primary-border hover:bg-primary-subtle peer-data-[state=checked]:border-primary peer-data-[state=checked]:bg-primary-subtle peer-focus-visible:ring-3 peer-focus-visible:ring-ring/50',
              showLabel ? 'gap-2 px-3' : 'w-9 p-1',
            )}
            htmlFor={id}
          >
            <span
              className={cn('brand-theme-swatch flex items-center justify-center rounded-sm shadow-xs', showLabel ? 'size-5' : 'size-7')}
              data-dark-foreground={brandColorUsesDarkForeground(preset)}
              style={{ backgroundColor: `var(--${preset}-9)` }}
            >
              <Check className={`size-4 transition-opacity ${checked ? 'opacity-100' : 'opacity-0'}`} />
            </span>
            <span className={showLabel ? 'text-sm font-medium text-foreground' : 'sr-only'}>{label}</span>
          </Label>
        </TooltipTrigger>
        <TooltipContent sideOffset={4}>
          {label}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
