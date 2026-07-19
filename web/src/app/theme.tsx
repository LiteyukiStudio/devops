import type { ReactNode } from 'react'
import type { ThemeContextValue, ThemeMode } from './theme-context'
import { useEffect, useMemo, useState } from 'react'
import { ThemeContext } from './theme-context'

const storageKey = 'luna-devops-theme'

function readStoredTheme(): ThemeMode {
  try {
    const stored = localStorage.getItem(storageKey)
    if (stored === 'light' || stored === 'dark' || stored === 'system')
      return stored
  }
  catch {
    // Fall through to the system preference when browser storage is unavailable.
  }
  return 'system'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(readStoredTheme)

  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => {
      const resolved = mode === 'system' ? (media.matches ? 'dark' : 'light') : mode
      document.documentElement.dataset.theme = resolved
      document.documentElement.classList.remove('light', 'dark')
      document.documentElement.classList.add(resolved)
      document.documentElement.style.colorScheme = resolved
    }

    apply()
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [mode])

  const value = useMemo<ThemeContextValue>(() => ({
    mode,
    setMode(nextMode) {
      try {
        localStorage.setItem(storageKey, nextMode)
      }
      catch {
        // The in-memory preference remains usable for the current session.
      }
      setMode(nextMode)
    },
  }), [mode])

  return <ThemeContext value={value}>{children}</ThemeContext>
}
