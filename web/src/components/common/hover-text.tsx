import type { ReactNode } from 'react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

/**
 * 单行文本展示组件。
 * 默认正常显示为单行省略，hover 时展示完整原文；适合错误摘要、资源名、路径等可能较长但不应撑开列表的文本。
 */
export function HoverText({
  children,
  className,
  contentClassName,
  side = 'top',
  value,
}: {
  children?: ReactNode
  className?: string
  contentClassName?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  value: string
}) {
  const content = value.trim()
  if (!content)
    return null

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn('block min-w-0 truncate', className)}>
          {children ?? content}
        </span>
      </TooltipTrigger>
      <TooltipContent className={cn('max-w-96 whitespace-pre-wrap break-words leading-5', contentClassName)} side={side}>
        {content}
      </TooltipContent>
    </Tooltip>
  )
}
