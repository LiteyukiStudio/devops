import type { ReactNode } from 'react'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'

interface EmptyStateProps {
  title: string
  description?: string
  actions?: ReactNode
  variant?: 'card' | 'plain'
}

/**
 * 列表、搜索结果和资源集合为空时的统一空状态。
 * 用于告诉用户当前没有数据并可附带创建/重试动作；加载中或接口失败分别使用 loading UI 和 ErrorState。
 */
export function EmptyState({ title, description, actions, variant = 'card' }: EmptyStateProps) {
  return (
    <Empty className={variant === 'plain' ? 'rounded-none border-0 bg-transparent shadow-none' : undefined}>
      <EmptyHeader>
        <EmptyTitle>{title}</EmptyTitle>
        {description && <EmptyDescription>{description}</EmptyDescription>}
      </EmptyHeader>
      {actions && <EmptyContent>{actions}</EmptyContent>}
    </Empty>
  )
}
