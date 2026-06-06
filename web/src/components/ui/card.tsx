import type { HTMLMotionProps } from 'motion/react'
import { motion } from 'motion/react'
import { cn } from '../../lib/utils'

export function Card({ className, ...props }: HTMLMotionProps<'div'>) {
  return (
    <motion.div
      className={cn('rounded-lg border border-border bg-surface p-4 shadow-sm transition-shadow duration-200', className)}
      transition={{ duration: 0.16, ease: 'easeOut' }}
      whileHover={{ y: -1 }}
      {...props}
    />
  )
}
