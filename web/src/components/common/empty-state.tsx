import type { ReactNode } from 'react'
import { Card } from '../ui/card'

export function EmptyState({ title, description, actions }: { title: string, description?: string, actions?: ReactNode }) {
  return (
    <Card className="text-sm text-muted-foreground">
      <p className="font-medium text-foreground">{title}</p>
      {description && <p className="mt-1">{description}</p>}
      {actions && <div className="mt-3">{actions}</div>}
    </Card>
  )
}
