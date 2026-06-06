import { AlertTriangle } from 'lucide-react'
import { AnimatePresence, motion } from 'motion/react'
import { Button } from '../ui/button'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: string
  confirmText?: string
  pending?: boolean
  onConfirm: () => void
  onOpenChange: (open: boolean) => void
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmText = '确认',
  pending = false,
  onConfirm,
  onOpenChange,
}: ConfirmDialogProps) {
  return (
    <AnimatePresence>
      {open && (
        <motion.div
          animate={{ opacity: 1 }}
          className="fixed inset-0 z-50 grid place-items-center bg-black/35 px-4"
          exit={{ opacity: 0 }}
          initial={{ opacity: 0 }}
          role="presentation"
          transition={{ duration: 0.16, ease: 'easeOut' }}
        >
          <motion.section
            aria-describedby="confirm-dialog-description"
            aria-labelledby="confirm-dialog-title"
            aria-modal="true"
            animate={{ opacity: 1, scale: 1, y: 0 }}
            className="w-full max-w-sm rounded-lg border border-border bg-surface p-4 shadow-xl"
            exit={{ opacity: 0, scale: 0.98, y: 4 }}
            initial={{ opacity: 0, scale: 0.98, y: 6 }}
            role="dialog"
            transition={{ duration: 0.18, ease: 'easeOut' }}
          >
            <div className="mb-4 flex gap-3">
              <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted text-danger">
                <AlertTriangle size={18} />
              </span>
              <div>
                <h2 id="confirm-dialog-title" className="font-semibold">{title}</h2>
                <p id="confirm-dialog-description" className="mt-1 text-sm text-muted-foreground">{description}</p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button disabled={pending} variant="secondary" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button disabled={pending} variant="danger" onClick={onConfirm}>
                {confirmText}
              </Button>
            </div>
          </motion.section>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
