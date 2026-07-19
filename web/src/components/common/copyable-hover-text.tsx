import type { ReactNode } from 'react'
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

export function CopyableHoverText({
  children,
  className,
  contentClassName,
  display,
  side = 'top',
  value,
}: {
  children?: ReactNode
  className?: string
  contentClassName?: string
  display?: ReactNode
  side?: 'top' | 'right' | 'bottom' | 'left'
  value?: string
}) {
  const { t } = useTranslation()
  const content = value?.trim() ?? ''
  const visible = display ?? children ?? content

  if (!content)
    return <span className={cn('block min-w-0 truncate', className)}>{visible || '-'}</span>

  const copyValue = () => {
    navigator.clipboard.writeText(content)
      .then(() => toast.success(t('common.copied')))
      .catch(error => toast.error(error.message))
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          className={cn('block min-w-0 truncate text-left transition hover:text-primary-text', className)}
          type="button"
          onClick={copyValue}
        >
          {visible}
        </button>
      </TooltipTrigger>
      <TooltipContent className={cn('flex max-w-96 items-start gap-2 break-all leading-5', contentClassName)} side={side}>
        <button
          aria-label={t('common.copy')}
          className="mt-0.5 inline-flex size-5 shrink-0 items-center justify-center rounded-sm text-background/80 transition hover:bg-background/15 hover:text-background"
          type="button"
          onClick={copyValue}
        >
          <Copy className="size-3.5" />
        </button>
        <span>{content}</span>
      </TooltipContent>
    </Tooltip>
  )
}
