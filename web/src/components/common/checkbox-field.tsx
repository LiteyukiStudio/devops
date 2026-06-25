import type { ComponentPropsWithRef, ReactNode } from 'react'
import { cn } from '@/lib/utils'

type CheckboxFieldProps = Omit<ComponentPropsWithRef<'input'>, 'children' | 'type'> & {
  children: ReactNode
  description?: ReactNode
  inputClassName?: string
}

export function CheckboxField({
  children,
  className,
  description,
  inputClassName,
  ref,
  ...props
}: CheckboxFieldProps) {
  return (
    <label className={cn('flex items-start gap-3 text-sm text-foreground', className)}>
      <input
        ref={ref}
        className={cn('mt-0.5 size-4 shrink-0 accent-primary', inputClassName)}
        type="checkbox"
        {...props}
      />
      <span className="min-w-0">
        <span className="block font-medium leading-5">{children}</span>
        {description && (
          <span className="mt-1 block text-xs leading-5 text-muted-foreground">
            {description}
          </span>
        )}
      </span>
    </label>
  )
}
