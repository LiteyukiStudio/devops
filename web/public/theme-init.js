(() => {
  const root = document.documentElement
  const validBrandPresets = new Set([
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
  const readStorage = (key) => {
    try {
      return localStorage.getItem(key)
    }
    catch {
      return null
    }
  }

  const mode = readStorage('luna-devops-theme')
  const resolvedMode = mode === 'light' || mode === 'dark'
    ? mode
    : window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'

  root.dataset.theme = resolvedMode
  root.classList.remove('light', 'dark')
  root.classList.add(resolvedMode)
  root.style.colorScheme = resolvedMode

  const injectedPreset = validBrandPresets.has(root.dataset.brandTheme) ? root.dataset.brandTheme : null
  const cachedSitePreset = readStorage('luna-devops-site-brand-color-preset')
  const sitePreset = injectedPreset || (validBrandPresets.has(cachedSitePreset) ? cachedSitePreset : 'blue')
  try {
    localStorage.setItem('luna-devops-site-brand-color-preset', sitePreset)
  }
  catch {
    // The injected site preset remains available when browser storage is unavailable.
  }

  const activeUserId = readStorage('luna-devops-theme-active-user')
  const userPreset = activeUserId
    ? readStorage(`luna-devops-user-brand-color-preset:${activeUserId}`)
    : null
  root.dataset.brandTheme = validBrandPresets.has(userPreset) ? userPreset : sitePreset

  const siteInterfaceStyle = readStorage('luna-devops-site-minimal-mode') === 'true'
    ? 'minimal'
    : 'themed'
  const activeInterfaceStyleUserId = readStorage('luna-devops-interface-style-active-user')
  const userInterfaceStyle = activeInterfaceStyleUserId
    ? readStorage(`luna-devops-user-interface-style:${activeInterfaceStyleUserId}`)
    : null
  root.dataset.interfaceStyle = userInterfaceStyle === 'minimal' || userInterfaceStyle === 'themed'
    ? userInterfaceStyle
    : siteInterfaceStyle
})()
