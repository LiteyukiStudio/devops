import type { ReactNode } from 'react'
import type { User } from '@/api'
import { Check, ChevronDown, Search, Users } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { UserAvatar } from '@/components/common/user-avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

const ALL_USERS_VALUE = '__all_users__'

interface UserSelectProps {
  allLabel?: string
  ariaLabel?: string
  className?: string
  disabled?: boolean
  emptyLabel?: string
  includeAll?: boolean
  placeholder: string
  users: User[]
  value: string
  onChange: (value: string) => void
}

export function UserSelect({
  allLabel,
  ariaLabel,
  className,
  disabled,
  emptyLabel,
  includeAll = false,
  placeholder,
  users,
  value,
  onChange,
}: UserSelectProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const selectedUser = users.find(user => user.id === value)
  const normalizedSearch = search.trim().toLowerCase()
  const filteredUsers = useMemo(() => {
    if (!normalizedSearch)
      return users.slice(0, 50)
    return users
      .filter(user => [user.name, user.email].some(text => text.toLowerCase().includes(normalizedSearch)))
      .slice(0, 50)
  }, [normalizedSearch, users])
  const selectedValue = value || (includeAll ? ALL_USERS_VALUE : '')
  const label = selectedUser ? userDisplayName(selectedUser) : includeAll && !value ? allLabel : placeholder

  function selectValue(nextValue: string) {
    onChange(nextValue === ALL_USERS_VALUE ? '' : nextValue)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          aria-label={ariaLabel ?? placeholder}
          aria-expanded={open}
          className={cn('h-11 w-full justify-between rounded-2xl px-4 font-normal', className)}
          disabled={disabled}
          type="button"
          variant="outline"
        >
          <span className={cn('flex min-w-0 items-center gap-2 text-left', !selectedUser && !(includeAll && !value) && 'text-muted-foreground')}>
            {selectedUser
              ? <UserAvatar className="size-6" user={selectedUser} />
              : includeAll && !value
                ? <Users className="size-5 shrink-0 text-muted-foreground" />
                : null}
            <span className="min-w-0 truncate">{label}</span>
          </span>
          <ChevronDown className={cn('ml-2 size-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')} />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="grid max-h-80 w-[var(--radix-popover-trigger-width)] min-w-72 grid-rows-[auto_minmax(0,1fr)] overflow-hidden p-0"
        sideOffset={6}
      >
        <div className="flex items-center gap-2 border-b border-border p-2">
          <Search className="size-4 shrink-0 text-muted-foreground" />
          <Input
            autoFocus
            className="h-8 rounded-md border-0 px-0 shadow-none focus-visible:ring-0"
            placeholder={t('common.search')}
            value={search}
            onChange={event => setSearch(event.target.value)}
          />
        </div>
        <div className="min-h-0 overflow-y-auto overscroll-contain p-1" onWheel={event => event.stopPropagation()}>
          {includeAll && !normalizedSearch && (
            <UserSelectOption
              checked={selectedValue === ALL_USERS_VALUE}
              description=""
              icon={<Users className="size-6 shrink-0 rounded-full bg-muted p-1 text-muted-foreground" />}
              label={allLabel ?? placeholder}
              value={ALL_USERS_VALUE}
              onSelect={selectValue}
            />
          )}
          {filteredUsers.length === 0 && (
            <p className="px-3 py-2 text-sm text-muted-foreground">{emptyLabel ?? t('common.noOptions')}</p>
          )}
          {filteredUsers.map(user => (
            <UserSelectOption
              key={user.id}
              checked={user.id === selectedValue}
              description={user.email}
              icon={<UserAvatar className="size-6" user={user} />}
              label={userDisplayName(user)}
              value={user.id}
              onSelect={selectValue}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}

function UserSelectOption({
  checked,
  description,
  icon,
  label,
  value,
  onSelect,
}: {
  checked: boolean
  description: string
  icon: ReactNode
  label: string
  value: string
  onSelect: (value: string) => void
}) {
  return (
    <button
      className="flex w-full min-w-0 items-center gap-2 rounded-md px-3 py-2 text-left text-sm hover:bg-muted"
      type="button"
      onClick={() => onSelect(value)}
    >
      {icon}
      <span className="min-w-0 flex-1">
        <span className="block truncate font-medium">{label}</span>
        {description && <span className="block truncate text-xs text-muted-foreground">{description}</span>}
      </span>
      {checked && <Check className="size-4 shrink-0 text-primary" />}
    </button>
  )
}

function userDisplayName(user: User) {
  return user.name || user.email
}
