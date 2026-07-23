import type { ReactNode } from 'react'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  title: string
  description?: ReactNode
  actions?: ReactNode
  icon?: ReactNode
  mode?: 'actionable' | 'filtered'
  variant?: 'card' | 'plain'
}

/**
 * 列表、搜索结果和资源集合为空时的统一空状态。
 * 用于告诉用户当前没有数据并可附带创建/重试动作；加载中或接口失败分别使用 loading UI 和 ErrorState。
 */
export function EmptyState({ title, description, actions, icon, mode = 'actionable', variant = 'card' }: EmptyStateProps) {
  const filtered = mode === 'filtered'
  return (
    <Empty
      className={cn(
        filtered ? 'min-h-24 items-center text-center' : 'min-h-32',
        variant === 'plain' && 'rounded-none border-0 bg-transparent shadow-none',
      )}
    >
      {icon && (
        <div className="mb-3 flex size-10 items-center justify-center rounded-md bg-muted text-muted-foreground" aria-hidden="true">
          {icon}
        </div>
      )}
      <EmptyHeader className={cn(filtered && 'justify-items-center')}>
        <EmptyTitle>{title}</EmptyTitle>
        {description && <EmptyDescription>{description}</EmptyDescription>}
      </EmptyHeader>
      {actions && <EmptyContent>{actions}</EmptyContent>}
    </Empty>
  )
}
