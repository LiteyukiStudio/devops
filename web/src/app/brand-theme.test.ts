import { beforeEach, describe, expect, it } from 'vitest'
import { applySiteBrandColorPreset, applyUserBrandColorPreference, brandColorPresets, brandColorUsesDarkForeground, brandThemeIsComposite, brandThemeSwatchBackground, brandThemeSwatchColors, clearActiveUserBrandColorPreference, defaultBrandColorPreset, normalizeBrandColorPreset, normalizeUserBrandColorPreference, themePickerPresets } from './brand-theme'

describe('brand theme presets', () => {
  beforeEach(() => {
    localStorage.clear()
    delete document.documentElement.dataset.brandTheme
  })

  it('keeps curated multi-color themes before the complete Radix scale list', () => {
    expect(brandColorPresets).toEqual([
      'aurora',
      'harbor',
      'sunset',
      'botanical',
      'meadow',
      'citrus',
      'gold',
      'bronze',
      'brown',
      'yellow',
      'amber',
      'orange',
      'tomato',
      'red',
      'ruby',
      'crimson',
      'pink',
      'plum',
      'purple',
      'violet',
      'iris',
      'indigo',
      'blue',
      'cyan',
      'teal',
      'jade',
      'green',
      'grass',
      'lime',
      'mint',
      'sky',
    ])
  })

  it('keeps the picker focused on composite themes and a small solid-color fallback set', () => {
    expect(themePickerPresets).toEqual([
      'aurora',
      'harbor',
      'sunset',
      'botanical',
      'meadow',
      'citrus',
      'gold',
      'orange',
      'red',
      'pink',
      'violet',
      'blue',
      'cyan',
      'teal',
      'green',
      'lime',
    ])
  })

  it('falls back to the default for unknown values', () => {
    expect(normalizeBrandColorPreset(' Teal ')).toBe('teal')
    expect(normalizeBrandColorPreset('custom-css')).toBe(defaultBrandColorPreset)
  })

  it('uses the Radix dark-foreground group for bright solid scales', () => {
    expect(brandColorPresets.filter(brandColorUsesDarkForeground)).toEqual(['yellow', 'amber', 'lime', 'mint', 'sky'])
  })

  it('describes composite and single-color swatches consistently', () => {
    expect(brandThemeIsComposite('aurora')).toBe(true)
    expect(brandThemeIsComposite('blue')).toBe(false)
    expect(brandThemeSwatchColors('aurora')).toEqual(['#3b6fe8', '#7867d9', '#2e9eaa', '#e7a63a'])
    expect(brandThemeSwatchColors('botanical')?.[0]).toBe('#0d4336')
    expect(brandThemeSwatchColors('blue')).toBeNull()
    expect(brandThemeSwatchBackground('aurora')).toBe('#3b6fe8')
    expect(brandThemeSwatchBackground('blue')).toBe('var(--blue-9)')
  })

  it('keeps an empty user preference as platform inheritance', () => {
    expect(normalizeUserBrandColorPreference('')).toBe('')
    expect(normalizeUserBrandColorPreference(' Ruby ')).toBe('ruby')
    expect(normalizeUserBrandColorPreference('custom-css')).toBe('')
  })

  it('applies user preference before site preference and restores the site preference', () => {
    applySiteBrandColorPreset('blue')
    applyUserBrandColorPreference('usr_theme', 'teal')
    expect(document.documentElement.dataset.brandTheme).toBe('teal')

    applySiteBrandColorPreset('ruby')
    expect(document.documentElement.dataset.brandTheme).toBe('teal')

    applyUserBrandColorPreference('usr_theme', '')
    expect(document.documentElement.dataset.brandTheme).toBe('ruby')

    clearActiveUserBrandColorPreference()
    expect(document.documentElement.dataset.brandTheme).toBe('ruby')
  })
})
