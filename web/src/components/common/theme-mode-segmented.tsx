import type { ThemeMode } from '../../app/theme-context'
import { Monitor, Moon, Sun } from 'lucide-react'
import { motion } from 'motion/react'
import { cn } from '../../lib/utils'

const modes: Array<{ value: ThemeMode, label: string, icon: typeof Sun }> = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'system', label: 'System', icon: Monitor },
  { value: 'dark', label: 'Dark', icon: Moon },
]

export function ThemeModeSegmented({ mode, setMode }: { mode: ThemeMode, setMode: (mode: ThemeMode) => void }) {
  return (
    <div className="relative grid grid-cols-3 gap-1 rounded-full bg-muted p-1">
      {modes.map((item) => {
        const Icon = item.icon
        const active = mode === item.value
        return (
          <button
            key={item.value}
            aria-label={item.label}
            aria-pressed={active}
            className={cn(
              'relative z-10 flex h-8 items-center justify-center rounded-full text-muted-foreground transition-colors duration-150',
              active && 'text-primary',
              !active && 'hover:bg-surface/70 hover:text-foreground',
            )}
            type="button"
            onClick={() => setMode(item.value)}
          >
            {active && (
              <motion.span
                className="absolute inset-0 -z-10 rounded-full bg-surface shadow-sm"
                layoutId="theme-mode-active"
                transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
              />
            )}
            <Icon size={15} />
          </button>
        )
      })}
    </div>
  )
}
