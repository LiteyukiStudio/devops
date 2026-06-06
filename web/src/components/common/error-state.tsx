import { AlertTriangle } from 'lucide-react'
import { Card } from '../ui/card'

export function ErrorState({ title, description }: { title: string, description?: string }) {
  return (
    <Card className="flex items-start gap-3 border-danger/30 bg-danger/5">
      <span className="mt-0.5 text-danger">
        <AlertTriangle size={18} />
      </span>
      <div>
        <p className="font-medium text-foreground">{title}</p>
        {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      </div>
    </Card>
  )
}
