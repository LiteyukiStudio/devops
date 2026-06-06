import type { CurrentUser } from '../../api/client'
import { LogOut } from 'lucide-react'
import { Button } from '../ui/button'

function initialsFromUser(user?: CurrentUser) {
  const source = user?.name || user?.email || 'U'
  return source.trim().slice(0, 2).toUpperCase()
}

export function SidebarUserPanel({
  user,
  logoutLabel,
  logoutPending,
  onLogout,
}: {
  user?: CurrentUser
  logoutLabel: string
  logoutPending?: boolean
  onLogout: () => void
}) {
  return (
    <div className="rounded-lg border border-border bg-background p-2">
      <div className="flex min-w-0 items-center gap-3">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground">
          {initialsFromUser(user)}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{user?.name ?? 'Demo User'}</p>
          <p className="truncate text-xs text-muted-foreground">{user?.email ?? 'demo@liteyuki.dev'}</p>
        </div>
        <Button aria-label={logoutLabel} className="size-8 px-0" disabled={logoutPending} variant="ghost" onClick={onLogout}>
          <LogOut size={15} />
        </Button>
      </div>
    </div>
  )
}
