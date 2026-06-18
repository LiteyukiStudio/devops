import type { ReactNode, UIEvent } from 'react'
import { useEffect, useRef } from 'react'
import { cn } from '@/lib/utils'

const bottomThreshold = 32

export function AutoFollowLog({
  className,
  content,
  emptyFallback,
  resetKey,
}: {
  className?: string
  content?: string
  emptyFallback: ReactNode
  resetKey?: string
}) {
  const viewportRef = useRef<HTMLPreElement>(null)
  const shouldFollowBottomRef = useRef(true)

  const scrollToBottom = () => {
    const viewport = viewportRef.current
    if (!viewport)
      return
    viewport.scrollTop = viewport.scrollHeight
  }

  const handleScroll = (event: UIEvent<HTMLPreElement>) => {
    const viewport = event.currentTarget
    shouldFollowBottomRef.current = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight <= bottomThreshold
  }

  useEffect(() => {
    shouldFollowBottomRef.current = true
    const frame = requestAnimationFrame(scrollToBottom)
    return () => cancelAnimationFrame(frame)
  }, [resetKey])

  useEffect(() => {
    if (!shouldFollowBottomRef.current)
      return
    const frame = requestAnimationFrame(scrollToBottom)
    return () => cancelAnimationFrame(frame)
  }, [content])

  return (
    <pre ref={viewportRef} className={cn('overflow-auto', className)} onScroll={handleScroll}>
      {content || emptyFallback}
    </pre>
  )
}
