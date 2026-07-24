import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { use } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { PageChromeTargetsContext } from '@/components/common/page-chrome-context'
import { cn } from '@/lib/utils'

interface PageChromeBackNavigation {
  label: string
  to: string
}

/**
 * 登录后页面的统一头部框架。
 * 第一行固定承载标题与页面工具；可选返回导航位于标题下方；存在 tabs 时才显示末行导航。
 */
export function PageChrome({
  backNavigation,
  tabsTargetRef,
  title,
  toolsTargetRef,
}: {
  backNavigation?: PageChromeBackNavigation
  tabsTargetRef: (node: HTMLDivElement | null) => void
  title: ReactNode
  toolsTargetRef: (node: HTMLDivElement | null) => void
}) {
  return (
    <div className={cn('min-w-0', !backNavigation && 'hidden lg:block')} data-slot="page-chrome">
      <div className="hidden min-w-0 flex-col gap-group lg:flex">
        <div className="flex min-h-10 min-w-0 items-center justify-between gap-section" data-slot="page-chrome-title-row">
          <div className="min-w-0 flex-1">{title}</div>
          <div ref={toolsTargetRef} className="flex min-w-0 shrink-0 items-center justify-end empty:hidden" />
        </div>
        {backNavigation && <BackNavigationLink {...backNavigation} />}
        <div ref={tabsTargetRef} className="min-w-0 empty:hidden" data-slot="page-chrome-tabs-row" />
      </div>
      {backNavigation && <BackNavigationLink className="lg:hidden" {...backNavigation} />}
    </div>
  )
}

function BackNavigationLink({
  className,
  label,
  to,
}: PageChromeBackNavigation & { className?: string }) {
  return (
    <Link
      className={cn(
        'inline-flex w-fit items-center gap-1.5 rounded-control px-1 py-1 text-sm font-medium text-muted-foreground outline-none transition-colors duration-fast ease-standard hover:bg-surface-subtle hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring',
        className,
      )}
      data-slot="page-chrome-back-navigation"
      to={to}
    >
      <ArrowLeft aria-hidden="true" className="size-4" />
      <span>{label}</span>
    </Link>
  )
}

/**
 * 将页面级操作放入 PageChrome 标题行；中小屏回落到正文流。
 */
export function PageChromeTools({ children, className }: { children?: ReactNode, className?: string }) {
  const { tools } = use(PageChromeTargetsContext)

  if (!children)
    return null

  return (
    <>
      {tools && createPortal(
        <div className={cn('flex min-w-0 flex-wrap items-center justify-end gap-2', className)}>
          {children}
        </div>,
        tools,
      )}
      <div className={cn('flex min-w-0 flex-wrap items-center gap-2 lg:hidden', className)}>
        {children}
      </div>
    </>
  )
}

/**
 * 将可选的页面级 Tab 放入 PageChrome 第二行；中小屏内容由调用方提供。
 */
export function PageChromeTabs({ children, className }: { children?: ReactNode, className?: string }) {
  const { tabs } = use(PageChromeTargetsContext)

  if (!children || !tabs)
    return null

  return createPortal(<div className={cn('min-w-0', className)}>{children}</div>, tabs)
}
