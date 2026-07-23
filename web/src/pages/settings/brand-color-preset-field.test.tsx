import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { TooltipProvider } from '@/components/ui/tooltip'
import i18next from '@/i18n'
import { BrandColorPresetField } from './brand-color-preset-field'

describe('brand color preset field', () => {
  beforeEach(async () => {
    await i18next.changeLanguage('en-US')
  })

  it('renders composite and single-color themes as stable circular radio options', () => {
    render(
      <TooltipProvider>
        <BrandColorPresetField
          ariaLabel="Color theme"
          inheritedPreset="aurora"
          inheritLabel="Follow platform"
          options={['aurora', 'blue']}
          value="aurora"
          onValueChange={vi.fn()}
        />
      </TooltipProvider>,
    )

    const aurora = screen.getByRole('radio', { name: 'Aurora' })
    const blue = screen.getByRole('radio', { name: 'Blue' })
    expect(aurora).toBeChecked()
    expect(blue).not.toBeChecked()

    const auroraSwatch = document.querySelector<HTMLSpanElement>('label[for="brand-color-aurora"] .brand-theme-swatch')
    const blueSwatch = document.querySelector<HTMLSpanElement>('label[for="brand-color-blue"] .brand-theme-swatch')
    const auroraGraphic = auroraSwatch?.querySelector<SVGElement>('[data-slot="composite-theme-swatch"]')
    expect(auroraSwatch).toHaveStyle({ background: '#3b6fe8' })
    expect(auroraGraphic?.querySelectorAll('path')).toHaveLength(3)
    expect(auroraGraphic?.querySelector('circle')).toHaveAttribute('fill', '#3b6fe8')
    expect(blueSwatch).toHaveStyle({ background: 'var(--blue-9)' })
    expect(blueSwatch?.querySelector('[data-slot="composite-theme-swatch"]')).not.toBeInTheDocument()
    expect(auroraSwatch).toHaveClass('rounded-full')
  })

  it('keeps platform inheritance as a dedicated radio option', () => {
    render(
      <TooltipProvider>
        <BrandColorPresetField
          ariaLabel="Color theme"
          inheritedPreset="harbor"
          inheritLabel="Follow platform"
          options={['aurora', 'blue']}
          value=""
          onValueChange={vi.fn()}
        />
      </TooltipProvider>,
    )

    expect(screen.getByRole('radio', { name: 'Follow platform' })).toBeChecked()
  })

  it('uses the curated picker set while keeping a stored legacy solid theme editable', () => {
    const { rerender } = render(
      <TooltipProvider>
        <BrandColorPresetField
          ariaLabel="Color theme"
          inheritedPreset="aurora"
          value="botanical"
          onValueChange={vi.fn()}
        />
      </TooltipProvider>,
    )

    expect(screen.getByRole('radio', { name: 'Botanical' })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'Blue' })).toBeInTheDocument()
    expect(screen.queryByRole('radio', { name: 'Ruby' })).not.toBeInTheDocument()

    rerender(
      <TooltipProvider>
        <BrandColorPresetField
          ariaLabel="Color theme"
          inheritedPreset="aurora"
          value="ruby"
          onValueChange={vi.fn()}
        />
      </TooltipProvider>,
    )

    expect(screen.getByRole('radio', { name: 'Ruby' })).toBeChecked()
  })
})
