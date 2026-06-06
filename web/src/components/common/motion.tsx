import type { HTMLMotionProps } from 'motion/react'
import { motion } from 'motion/react'

const easeOut = [0.16, 1, 0.3, 1] as const

const gentleTransition = {
  duration: 0.2,
  ease: easeOut,
}

export function PageMotion(props: HTMLMotionProps<'div'>) {
  return (
    <motion.div
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 6 }}
      initial={{ opacity: 0, y: 8 }}
      transition={gentleTransition}
      {...props}
    />
  )
}

export function MotionList(props: HTMLMotionProps<'div'>) {
  return (
    <motion.div
      animate="show"
      initial="hidden"
      variants={{
        hidden: {},
        show: {
          transition: {
            staggerChildren: 0.035,
          },
        },
      }}
      {...props}
    />
  )
}

export function MotionItem(props: HTMLMotionProps<'div'>) {
  return (
    <motion.div
      variants={{
        hidden: { opacity: 0, y: 8 },
        show: { opacity: 1, y: 0, transition: gentleTransition },
      }}
      {...props}
    />
  )
}
