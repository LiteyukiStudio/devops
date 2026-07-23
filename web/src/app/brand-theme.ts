export const defaultBrandColorPreset = 'blue'
export const siteBrandColorPresetStorageKey = 'luna-devops-site-brand-color-preset'
export const activeThemeUserStorageKey = 'luna-devops-theme-active-user'
export const userBrandColorPresetStoragePrefix = 'luna-devops-user-brand-color-preset:'

export const compositeBrandColorPresets = ['aurora', 'harbor', 'sunset', 'botanical', 'meadow', 'citrus'] as const

// Curated multi-color themes lead the official Radix single-color scales.
export const brandColorPresets = [
  ...compositeBrandColorPresets,
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
] as const

export type BrandColorPreset = typeof brandColorPresets[number]
export type UserBrandColorPreference = BrandColorPreset | ''

export const themePickerPresets: BrandColorPreset[] = [
  ...compositeBrandColorPresets,
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
]

const brandColorPresetSet = new Set<string>(brandColorPresets)
const compositeBrandColorPresetSet = new Set<string>(compositeBrandColorPresets)

export function normalizeBrandColorPreset(value: unknown): BrandColorPreset {
  const normalized = String(value ?? '').trim().toLowerCase()
  return brandColorPresetSet.has(normalized) ? normalized as BrandColorPreset : defaultBrandColorPreset
}

export function normalizeUserBrandColorPreference(value: unknown): UserBrandColorPreference {
  const normalized = String(value ?? '').trim().toLowerCase()
  return brandColorPresetSet.has(normalized) ? normalized as BrandColorPreset : ''
}

export function applyBrandColorPreset(value: unknown) {
  const preset = normalizeBrandColorPreset(value)
  document.documentElement.dataset.brandTheme = preset
  return preset
}

export function applySiteBrandColorPreset(value: unknown) {
  const sitePreset = normalizeBrandColorPreset(value)
  writeStorage(siteBrandColorPresetStorageKey, sitePreset)
  return applyBrandColorPreset(readActiveUserBrandColorPreference() || sitePreset)
}

export function applyUserBrandColorPreference(userId: string, value: unknown) {
  const normalizedUserId = userId.trim()
  const preference = normalizeUserBrandColorPreference(value)
  if (!normalizedUserId)
    return applyBrandColorPreset(readSiteBrandColorPreset())

  writeStorage(activeThemeUserStorageKey, normalizedUserId)
  if (preference)
    writeStorage(userBrandColorPresetStorageKey(normalizedUserId), preference)
  else
    removeStorage(userBrandColorPresetStorageKey(normalizedUserId))

  return applyBrandColorPreset(preference || readSiteBrandColorPreset())
}

export function clearActiveUserBrandColorPreference() {
  removeStorage(activeThemeUserStorageKey)
  return applyBrandColorPreset(readSiteBrandColorPreset())
}

export function userBrandColorPresetStorageKey(userId: string) {
  return `${userBrandColorPresetStoragePrefix}${userId}`
}

export function brandColorUsesDarkForeground(preset: BrandColorPreset) {
  return preset === 'sky' || preset === 'mint' || preset === 'lime' || preset === 'yellow' || preset === 'amber'
}

export function brandThemeIsComposite(preset: BrandColorPreset) {
  return compositeBrandColorPresetSet.has(preset)
}

export function brandThemeSwatchBackground(preset: BrandColorPreset) {
  switch (preset) {
    case 'aurora':
      return 'conic-gradient(from 45deg, #3b6fe8 0 25%, #7867d9 25% 50%, #2e9eaa 50% 75%, #e7a63a 75% 100%)'
    case 'harbor':
      return 'conic-gradient(from 45deg, #147d73 0 34%, #47709b 34% 67%, #ce8a62 67% 100%)'
    case 'sunset':
      return 'conic-gradient(from 45deg, #eb7a53 0 34%, #f7d44c 34% 67%, #7c5cc4 67% 100%)'
    case 'botanical':
      return 'conic-gradient(from 45deg, #0d4336 0 34%, #ce8a62 34% 67%, #f4f2f1 67% 100%)'
    case 'meadow':
      return 'conic-gradient(from 45deg, #5cab8c 0 34%, #fbfe8d 34% 67%, #d4d5cf 67% 100%)'
    case 'citrus':
      return 'conic-gradient(from 45deg, #eb7a53 0 34%, #f7d44c 34% 67%, #f6ecc9 67% 100%)'
    default:
      return `var(--${preset}-9)`
  }
}

function readActiveUserBrandColorPreference() {
  const userId = readStorage(activeThemeUserStorageKey)
  return userId ? normalizeUserBrandColorPreference(readStorage(userBrandColorPresetStorageKey(userId))) : ''
}

function readSiteBrandColorPreset() {
  return normalizeBrandColorPreset(readStorage(siteBrandColorPresetStorageKey))
}

function readStorage(key: string) {
  try {
    return localStorage.getItem(key)
  }
  catch {
    return null
  }
}

function writeStorage(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  }
  catch {
    // The current DOM theme remains usable when browser storage is unavailable.
  }
}

function removeStorage(key: string) {
  try {
    localStorage.removeItem(key)
  }
  catch {
    // The current DOM theme remains usable when browser storage is unavailable.
  }
}
