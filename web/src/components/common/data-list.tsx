import type { ReactNode } from 'react'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { EmptyState } from './empty-state'
import { PaginationController } from './pagination'

export interface DataListColumn<T> {
  key: string
  header: ReactNode
  className?: string
  headerClassName?: string
  cellClassName?: string
  render: (item: T) => ReactNode
}

interface DataListProps<T> {
  items: T[]
  columns: DataListColumn<T>[]
  rowKey: (item: T) => string
  title?: ReactNode
  variant?: 'card' | 'plain'
  emptyTitle: string
  emptyDescription?: string
  search?: {
    value: string
    placeholder: string
    onChange: (value: string) => void
  }
  selection?: {
    selectedKeys: string[]
    selectAllLabel: string
    selectRowLabel: (item: T) => string
    selectedLabel: string
    isRowSelectable?: (item: T) => boolean
    bulkActions?: ReactNode
    onSelectionChange: (keys: string[]) => void
  }
  pagination?: {
    page: number
    pageSize: number
    defaultPageSize?: number
    total: number
    totalPages: number
    pageInfoLabel: string
    onPageChange: (page: number) => void
    onPageSizeChange?: (pageSize: number) => void
    pageSizeOptions?: number[]
  }
}

/**
 * 管理台列表和表格的统一展示组件。
 * 用于资源列表、用户列表、凭据列表等需要列、空状态和分页的场景；布局型页面或少量指标卡片不应套用它。
 */
export function DataList<T>({
  items,
  columns,
  rowKey,
  title,
  variant = 'card',
  emptyTitle,
  emptyDescription,
  search,
  selection,
  pagination,
}: DataListProps<T>) {
  const selectedKeySet = new Set(selection?.selectedKeys ?? [])
  const rowKeys = items.map(rowKey)
  const selectableRowKeys = selection ? items.filter(item => selection.isRowSelectable?.(item) ?? true).map(rowKey) : rowKeys
  const selectable = Boolean(selection)
  const allRowsSelected = selectableRowKeys.length > 0 && selectableRowKeys.every(key => selectedKeySet.has(key))
  const someRowsSelected = selectableRowKeys.some(key => selectedKeySet.has(key))
  const updateRowSelection = (key: string, selected: boolean) => {
    if (!selection)
      return
    const next = new Set(selection.selectedKeys)
    if (selected)
      next.add(key)
    else
      next.delete(key)
    selection.onSelectionChange([...next])
  }
  const updateAllRowsSelection = (selected: boolean) => {
    if (!selection)
      return
    const next = new Set(selection.selectedKeys)
    for (const key of selectableRowKeys) {
      if (selected)
        next.add(key)
      else
        next.delete(key)
    }
    selection.onSelectionChange([...next])
  }

  return (
    <Card className={cn('flex w-full min-w-0 max-w-full max-h-none flex-col overflow-hidden p-0 md:max-h-[calc(100vh-15rem)]', variant === 'plain' && 'rounded-none border-0 bg-transparent shadow-none')}>
      {(title || search || selection?.bulkActions) && (
        <div className="flex shrink-0 flex-col gap-3 border-b border-border px-4 py-4 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
          <div className="min-w-0">
            {title && <h2 className="text-base font-semibold">{title}</h2>}
            {selection && selection.selectedKeys.length > 0 && (
              <p className="mt-1 text-xs text-muted-foreground">{selection.selectedLabel}</p>
            )}
          </div>
          <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            {search && (
              <Input
                className="h-9 w-full sm:w-64"
                placeholder={search.placeholder}
                value={search.value}
                onChange={event => search.onChange(event.target.value)}
              />
            )}
            {selection?.bulkActions}
          </div>
        </div>
      )}
      <div className="min-h-0 w-full min-w-0 max-w-full flex-1 overflow-auto">
        {items.length === 0
          ? <EmptyState description={emptyDescription} title={emptyTitle} variant="plain" />
          : (
              <table className="w-full min-w-[56rem] table-auto caption-bottom text-sm" data-slot="data-list-table">
                <thead className="sticky top-0 z-10 bg-muted/95 backdrop-blur [&_tr]:border-b">
                  <tr className="border-b border-border transition-colors hover:bg-muted/40">
                    {selectable && (
                      <th className="h-10 w-10 px-4 py-3 text-left align-middle text-xs font-medium whitespace-nowrap text-muted-foreground">
                        <input
                          aria-label={selection?.selectAllLabel}
                          checked={allRowsSelected}
                          className="size-4 accent-primary"
                          disabled={selectableRowKeys.length === 0}
                          ref={(element) => {
                            if (element)
                              element.indeterminate = someRowsSelected && !allRowsSelected
                          }}
                          type="checkbox"
                          onChange={event => updateAllRowsSelection(event.target.checked)}
                        />
                      </th>
                    )}
                    {columns.map(column => (
                      <th
                        key={column.key}
                        className={cn(
                          'h-10 px-4 py-3 text-left align-middle text-xs font-medium whitespace-nowrap text-muted-foreground',
                          column.className,
                          column.headerClassName,
                        )}
                      >
                        {column.header}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="[&_tr:last-child]:border-0">
                  {items.map((item) => {
                    const itemKey = rowKey(item)
                    const rowSelectable = selection?.isRowSelectable?.(item) ?? true
                    return (
                      <tr key={itemKey} className="border-b border-border transition-colors hover:bg-muted/40">
                        {selectable && (
                          <td className="w-10 px-4 py-3 align-middle">
                            <input
                              aria-label={selection?.selectRowLabel(item)}
                              checked={selectedKeySet.has(itemKey)}
                              className="size-4 accent-primary"
                              disabled={!rowSelectable}
                              type="checkbox"
                              onChange={event => updateRowSelection(itemKey, event.target.checked)}
                            />
                          </td>
                        )}
                        {columns.map(column => (
                          <td key={column.key} className={cn('px-4 py-3 align-middle', column.className, column.cellClassName)}>
                            {column.render(item)}
                          </td>
                        ))}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
      </div>

      {pagination && (
        <div className="shrink-0 border-t border-border px-4 py-3 text-sm text-muted-foreground">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span>{pagination.pageInfoLabel}</span>
            <PaginationController
              initialPage={pagination.page}
              pageSize={pagination.pageSize}
              defaultPageSize={pagination.defaultPageSize}
              pageSizeOptions={pagination.pageSizeOptions}
              total={pagination.total}
              onPageChange={pagination.onPageChange}
              onPageSizeChange={pagination.onPageSizeChange}
            />
          </div>
        </div>
      )}
    </Card>
  )
}
