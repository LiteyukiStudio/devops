import type { ComponentProps } from 'react'

import { cn } from '@/lib/utils'
import { ScrollArea } from './scroll-area'

function Table({ className, ...props }: ComponentProps<'table'>) {
  return (
    <ScrollArea className="w-full" data-slot="table-container" scrollbars="horizontal" type="auto">
      <table className={cn('w-max min-w-full caption-bottom text-sm', className)} data-slot="table" {...props} />
    </ScrollArea>
  )
}

function TableHeader({ className, ...props }: ComponentProps<'thead'>) {
  return <thead className={cn('[&_tr]:border-b', className)} data-slot="table-header" {...props} />
}

function TableBody({ className, ...props }: ComponentProps<'tbody'>) {
  return <tbody className={cn('[&_tr:last-child]:border-0', className)} data-slot="table-body" {...props} />
}

function TableRow({ className, ...props }: ComponentProps<'tr'>) {
  return (
    <tr
      className={cn('border-b border-border transition-colors hover:bg-muted/40', className)}
      data-slot="table-row"
      {...props}
    />
  )
}

function TableHead({ className, ...props }: ComponentProps<'th'>) {
  return (
    <th
      className={cn('h-10 px-4 text-left align-middle text-xs font-medium whitespace-nowrap text-muted-foreground', className)}
      data-slot="table-head"
      {...props}
    />
  )
}

function TableCell({ className, ...props }: ComponentProps<'td'>) {
  return <td className={cn('px-4 py-3 align-middle', className)} data-slot="table-cell" {...props} />
}

export { Table, TableBody, TableCell, TableHead, TableHeader, TableRow }
