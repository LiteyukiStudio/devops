import type { ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface ProgressiveSectionProps {
  children: ReactNode
  defaultOpen?: boolean
  description?: ReactNode
  storageKey?: string
  summary?: ReactNode
  title: ReactNode
}

/**
 * 渐进式表单分组。
 * 用于把高级配置折叠到稳定摘要后面，避免首屏暴露过多字段；不负责表单状态和校验。
 */
export function ProgressiveSection({ children, defaultOpen = false, description, storageKey, summary, title }: ProgressiveSectionProps) {
  const [open, setOpen] = useState(() => {
    if (!storageKey || typeof window === 'undefined')
      return defaultOpen
    const stored = window.localStorage.getItem(storageKey)
    return stored === null ? defaultOpen : stored === 'true'
  })

  const toggleOpen = () => {
    setOpen((value) => {
      const next = !value
      if (storageKey && typeof window !== 'undefined')
        window.localStorage.setItem(storageKey, String(next))
      return next
    })
  }

  return (
    <section className="rounded-lg border border-border bg-card">
      <Button
        aria-expanded={open}
        className="flex h-auto w-full justify-between gap-3 rounded-lg px-4 py-3 text-left hover:bg-muted/60"
        type="button"
        variant="ghost"
        onClick={toggleOpen}
      >
        <span className="min-w-0">
          <span className="block text-sm font-semibold text-foreground">{title}</span>
          {summary && <span className="mt-1 block truncate text-xs text-muted-foreground">{summary}</span>}
          {description && open && <span className="mt-1 block text-xs text-muted-foreground">{description}</span>}
        </span>
        <ChevronDown className={cn('mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')} />
      </Button>
      {open && (
        <div className="grid gap-4 border-t border-border px-4 py-4">
          {children}
        </div>
      )}
    </section>
  )
}
