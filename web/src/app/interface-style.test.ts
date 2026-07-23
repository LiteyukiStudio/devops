import { beforeEach, describe, expect, it } from 'vitest'
import { applySiteMinimalModeDefault, applyUserInterfaceStylePreference, clearActiveUserInterfaceStylePreference, normalizeUserInterfaceStylePreference } from './interface-style'

describe('interface style preference', () => {
  beforeEach(() => {
    localStorage.clear()
    delete document.documentElement.dataset.interfaceStyle
  })

  it('normalizes only supported user preferences', () => {
    expect(normalizeUserInterfaceStylePreference('minimal')).toBe('minimal')
    expect(normalizeUserInterfaceStylePreference('themed')).toBe('themed')
    expect(normalizeUserInterfaceStylePreference('custom')).toBe('')
  })

  it('applies user preference before the site default', () => {
    applySiteMinimalModeDefault('true')
    expect(document.documentElement.dataset.interfaceStyle).toBe('minimal')

    applyUserInterfaceStylePreference('usr_style', 'themed')
    expect(document.documentElement.dataset.interfaceStyle).toBe('themed')

    applySiteMinimalModeDefault('false')
    expect(document.documentElement.dataset.interfaceStyle).toBe('themed')

    applyUserInterfaceStylePreference('usr_style', '')
    expect(document.documentElement.dataset.interfaceStyle).toBe('themed')

    applySiteMinimalModeDefault('true')
    expect(document.documentElement.dataset.interfaceStyle).toBe('minimal')

    clearActiveUserInterfaceStylePreference()
    expect(document.documentElement.dataset.interfaceStyle).toBe('minimal')
  })
})
