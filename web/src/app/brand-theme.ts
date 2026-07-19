export const defaultBrandColorPreset = 'blue'
export const siteBrandColorPresetStorageKey = 'luna-devops-site-brand-color-preset'
export const activeThemeUserStorageKey = 'luna-devops-theme-active-user'
export const userBrandColorPresetStoragePrefix = 'luna-devops-user-brand-color-preset:'

// Keep this order aligned with the official Radix Colors palette composition guide.
export const brandColorPresets = [
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

const brandColorPresetSet = new Set<string>(brandColorPresets)

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
