import type { DragEvent, ReactNode } from 'react'
import { useMemo, useState } from 'react'
import { cn } from '@/lib/utils'

interface SortableGridRenderState {
  dragging: boolean
  insertBefore: boolean
}

interface SortableGridProps<T> {
  className?: string
  getItemId: (item: T) => string
  items: T[]
  renderItem: (item: T, state: SortableGridRenderState) => ReactNode
  onOrderChange: (items: T[]) => void
}

export function SortableGrid<T>({ className, getItemId, items, onOrderChange, renderItem }: SortableGridProps<T>) {
  const [draggedId, setDraggedId] = useState('')
  const [insertIndex, setInsertIndex] = useState<number | null>(null)
  const orderedItems = useMemo(() => reorderedItemsForDrag(items, draggedId, insertIndex, getItemId), [draggedId, getItemId, insertIndex, items])

  const finishDrag = () => {
    if (!draggedId || insertIndex === null) {
      setDraggedId('')
      setInsertIndex(null)
      return
    }
    const nextItems = reorderedItemsForDrag(items, draggedId, insertIndex, getItemId)
    setDraggedId('')
    setInsertIndex(null)
    if (nextItems.map(getItemId).join(',') !== items.map(getItemId).join(',')) {
      onOrderChange(nextItems)
    }
  }

  return (
    <div className={cn('grid gap-3 sm:grid-cols-2 xl:grid-cols-4', className)}>
      {orderedItems.map((item, index) => {
        const itemId = getItemId(item)
        return (
          <div
            key={itemId}
            draggable
            onDragEnd={finishDrag}
            onDragEnter={(event) => {
              event.preventDefault()
              setInsertIndex(index)
            }}
            onDragOver={event => event.preventDefault()}
            onDragStart={(event: DragEvent<HTMLDivElement>) => {
              setDraggedId(itemId)
              setInsertIndex(index)
              event.dataTransfer.effectAllowed = 'move'
              event.dataTransfer.setData('text/plain', itemId)
            }}
            onDrop={(event) => {
              event.preventDefault()
              finishDrag()
            }}
          >
            {renderItem(item, {
              dragging: draggedId === itemId,
              insertBefore: draggedId !== '' && insertIndex === index,
            })}
          </div>
        )
      })}
    </div>
  )
}

function reorderedItemsForDrag<T>(items: T[], draggedId: string, insertIndex: number | null, getItemId: (item: T) => string) {
  if (!draggedId || insertIndex === null)
    return items
  const currentIndex = items.findIndex(item => getItemId(item) === draggedId)
  if (currentIndex < 0)
    return items
  const next = [...items]
  const [draggedItem] = next.splice(currentIndex, 1)
  const normalizedIndex = Math.max(0, Math.min(insertIndex, next.length))
  next.splice(normalizedIndex, 0, draggedItem)
  return next
}
