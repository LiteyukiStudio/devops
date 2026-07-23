import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type NoticeTone = 'danger' | 'info' | 'success' | 'warning'
type NoticeVariant = 'neutral' | 'tinted'

/** 页面级状态提示；状态颜色固定，不随个人品牌主题变化。 */
export function Notice({ actions, children, className, icon, title, tone = 'info', variant = 'tinted' }: {
  actions?: ReactNode
  children?: ReactNode
  className?: string
  icon?: ReactNode
  title: ReactNode
  tone?: NoticeTone
  variant?: NoticeVariant
}) {
  return (
    <div
      className={cn('grid gap-field rounded-container p-group', variant === 'neutral' ? 'bg-surface-raised' : noticeToneClassName(tone), className)}
      data-slot="notice"
      data-variant={variant}
      role="status"
    >
      <div className="flex min-w-0 flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-2">
          {icon && <span className={cn('mt-0.5 shrink-0', variant === 'neutral' && noticeToneTextClassName(tone))}>{icon}</span>}
          <div className="min-w-0">
            <h2 className="font-semibold text-foreground">{title}</h2>
            {children && <div className="mt-1 text-sm text-muted-foreground">{children}</div>}
          </div>
        </div>
        {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
      </div>
    </div>
  )
}

function noticeToneClassName(tone: NoticeTone) {
  if (tone === 'danger')
    return 'bg-danger-subtle'
  if (tone === 'warning')
    return 'bg-warning-subtle'
  if (tone === 'success')
    return 'bg-success-subtle'
  return 'bg-info-subtle'
}

function noticeToneTextClassName(tone: NoticeTone) {
  if (tone === 'danger')
    return 'text-danger'
  if (tone === 'warning')
    return 'text-warning'
  if (tone === 'success')
    return 'text-success'
  return 'text-info'
}
