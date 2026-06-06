import type { ReactNode } from 'react'
import { Badge } from '../ui/status'

export function StatusBadge({ children }: { children: ReactNode }) {
  return <Badge>{children}</Badge>
}
