import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn('h-9 w-full rounded-md border border-border bg-background px-3 text-sm outline-none transition focus:border-primary', className)}
      {...props}
    />
  )
}

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className={cn('min-h-20 w-full rounded-md border border-border bg-background px-3 py-2 text-sm outline-none transition focus:border-primary', className)}
      {...props}
    />
  )
}

export function Select({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={cn('h-9 w-full rounded-md border border-border bg-background px-3 text-sm outline-none transition focus:border-primary', className)}
      {...props}
    />
  )
}

export function Field({ label, children }: { label: string, children: ReactNode }) {
  return (
    <label className="grid gap-1.5 text-sm font-medium text-foreground">
      <span>{label}</span>
      {children}
    </label>
  )
}
