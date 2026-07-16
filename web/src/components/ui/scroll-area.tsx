import type { ComponentProps } from 'react'
import { ScrollArea as ScrollAreaPrimitive } from 'radix-ui'

import { cn } from '@/lib/utils'

type Scrollbars = 'both' | 'horizontal' | 'vertical'

function ScrollArea({
  className,
  children,
  scrollbars = 'vertical',
  type = 'hover',
  ...props
}: ComponentProps<typeof ScrollAreaPrimitive.Root> & { scrollbars?: Scrollbars }) {
  return (
    <ScrollAreaPrimitive.Root
      className={cn('relative overflow-hidden', className)}
      data-scroll-area-type={type}
      data-scrollbars={scrollbars}
      data-slot="scroll-area"
      type={type}
      {...props}
    >
      <ScrollAreaPrimitive.Viewport
        className="size-full rounded-[inherit] outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        data-slot="scroll-area-viewport"
      >
        {children}
      </ScrollAreaPrimitive.Viewport>
      {(scrollbars === 'vertical' || scrollbars === 'both') && <ScrollBar />}
      {(scrollbars === 'horizontal' || scrollbars === 'both') && <ScrollBar orientation="horizontal" />}
      <ScrollAreaPrimitive.Corner className="bg-muted" />
    </ScrollAreaPrimitive.Root>
  )
}

function ScrollBar({
  className,
  orientation = 'vertical',
  ...props
}: ComponentProps<typeof ScrollAreaPrimitive.Scrollbar>) {
  return (
    <ScrollAreaPrimitive.Scrollbar
      className={cn(
        'flex touch-none bg-muted/70 p-px transition-colors select-none',
        orientation === 'vertical' && 'h-full w-2.5 border-l border-l-transparent',
        orientation === 'horizontal' && 'h-2.5 flex-col border-t border-t-transparent',
        className,
      )}
      data-slot="scroll-area-scrollbar"
      orientation={orientation}
      {...props}
    >
      <ScrollAreaPrimitive.Thumb
        className="relative flex-1 rounded-full bg-muted-foreground/40 hover:bg-muted-foreground/60"
        data-slot="scroll-area-thumb"
      />
    </ScrollAreaPrimitive.Scrollbar>
  )
}

export { ScrollArea, ScrollBar }
