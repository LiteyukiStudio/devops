export type InterfaceStyle = 'minimal' | 'themed'
export type UserInterfaceStylePreference = '' | InterfaceStyle

const siteMinimalModeStorageKey = 'luna-devops-site-minimal-mode'
const activeUserStorageKey = 'luna-devops-interface-style-active-user'
const userStyleStoragePrefix = 'luna-devops-user-interface-style:'

export function applySiteMinimalModeDefault(value: unknown) {
  const minimal = String(value).trim().toLowerCase() === 'true'
  writeStorage(siteMinimalModeStorageKey, String(minimal))
  return applyInterfaceStyle(readActiveUserPreference() || (minimal ? 'minimal' : 'themed'))
}

export function applyUserInterfaceStylePreference(userId: string, value: unknown) {
  const normalizedUserId = userId.trim()
  const preference = normalizeUserInterfaceStylePreference(value)
  if (!normalizedUserId)
    return applyInterfaceStyle(readSiteDefault())

  writeStorage(activeUserStorageKey, normalizedUserId)
  if (preference)
    writeStorage(userStorageKey(normalizedUserId), preference)
  else
    removeStorage(userStorageKey(normalizedUserId))

  return applyInterfaceStyle(preference || readSiteDefault())
}

export function clearActiveUserInterfaceStylePreference() {
  removeStorage(activeUserStorageKey)
  return applyInterfaceStyle(readSiteDefault())
}

export function normalizeUserInterfaceStylePreference(value: unknown): UserInterfaceStylePreference {
  return value === 'minimal' || value === 'themed' ? value : ''
}

function applyInterfaceStyle(style: InterfaceStyle) {
  document.documentElement.dataset.interfaceStyle = style
  return style
}

function readActiveUserPreference() {
  const userId = readStorage(activeUserStorageKey)
  return userId ? normalizeUserInterfaceStylePreference(readStorage(userStorageKey(userId))) : ''
}

function readSiteDefault(): InterfaceStyle {
  return readStorage(siteMinimalModeStorageKey) === 'true' ? 'minimal' : 'themed'
}

function userStorageKey(userId: string) {
  return `${userStyleStoragePrefix}${userId}`
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
    // The current DOM style remains usable when browser storage is unavailable.
  }
}

function removeStorage(key: string) {
  try {
    localStorage.removeItem(key)
  }
  catch {
    // The current DOM style remains usable when browser storage is unavailable.
  }
}
