import type { MFAEnrollment } from '@/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, KeyRound, RefreshCw, ShieldCheck, ShieldOff } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api'
import { ConfirmDialog } from '@/components/common/confirm-dialog'
import { ErrorState } from '@/components/common/error-state'
import { FormField as Field } from '@/components/common/form-field'
import { StatusBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

const mfaStatusQueryKey = ['mfa-status'] as const

export function AccountMFAPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [enrollment, setEnrollment] = useState<MFAEnrollment>()
  const [confirmationCode, setConfirmationCode] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const status = useQuery({ queryKey: mfaStatusQueryKey, queryFn: api.getMFAStatus })

  const enroll = useMutation({
    mutationFn: api.enrollMFA,
    onSuccess: (result) => {
      setConfirmationCode('')
      setEnrollment(result)
    },
    onError: error => toast.error(error.message),
  })

  const confirm = useMutation({
    mutationFn: api.confirmMFAEnrollment,
    onSuccess: (result) => {
      setEnrollment(undefined)
      setConfirmationCode('')
      setRecoveryCodes(result.recoveryCodes)
      queryClient.invalidateQueries({ queryKey: mfaStatusQueryKey })
      toast.success(t('accountPage.mfa.enabledToast'))
    },
    onError: error => toast.error(error.message),
  })

  const regenerate = useMutation({
    mutationFn: api.regenerateMFARecoveryCodes,
    onSuccess: (result) => {
      setRecoveryCodes(result.recoveryCodes)
      queryClient.invalidateQueries({ queryKey: mfaStatusQueryKey })
      toast.success(t('accountPage.mfa.recoveryRegenerated'))
    },
    onError: error => toast.error(error.message),
  })

  const disable = useMutation({
    mutationFn: api.disableMFA,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mfaStatusQueryKey })
      toast.success(t('accountPage.mfa.disabledToast'))
    },
    onError: error => toast.error(error.message),
  })

  const copyRecoveryCodes = () => {
    navigator.clipboard.writeText(recoveryCodes.join('\n'))
      .then(() => toast.success(t('common.copied')))
      .catch(error => toast.error(error.message))
  }

  return (
    <>
      <Card className="grid gap-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <ShieldCheck size={18} />
            </span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-base font-semibold">{t('accountPage.mfa.title')}</h2>
                {status.data && (
                  <StatusBadge tone={status.data.enabled ? 'success' : 'neutral'}>
                    {status.data.enabled ? t('common.enabled') : t('common.disabled')}
                  </StatusBadge>
                )}
              </div>
              <p className="mt-1 text-sm text-muted-foreground">{t('accountPage.mfa.description')}</p>
            </div>
          </div>
          {!status.data?.enabled && (
            <Button disabled={status.isLoading || status.isError || enroll.isPending} onClick={() => enroll.mutate()}>
              <KeyRound size={16} />
              {t('accountPage.mfa.enable')}
            </Button>
          )}
        </div>

        {status.isError && <ErrorState title={t('accountPage.mfa.loadFailedTitle')} description={t('accountPage.mfa.loadFailedDescription')} />}
        {status.data?.enabled && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-4">
            <p className="text-sm text-muted-foreground">
              {t('accountPage.mfa.recoveryRemaining', { count: status.data.recoveryCodesRemaining })}
            </p>
            <div className="flex flex-wrap gap-2">
              <Button disabled={regenerate.isPending} variant="outline" onClick={() => regenerate.mutate()}>
                <RefreshCw size={16} />
                {t('accountPage.mfa.regenerateRecovery')}
              </Button>
              <ConfirmDialog
                confirmText={t('accountPage.mfa.disable')}
                description={t('accountPage.mfa.disableDescription')}
                pending={disable.isPending}
                title={t('accountPage.mfa.disableTitle')}
                onConfirm={() => disable.mutateAsync()}
              >
                <Button variant="destructive">
                  <ShieldOff size={16} />
                  {t('accountPage.mfa.disable')}
                </Button>
              </ConfirmDialog>
            </div>
          </div>
        )}
      </Card>

      <Dialog open={Boolean(enrollment)} onOpenChange={open => !open && setEnrollment(undefined)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('accountPage.mfa.enrollTitle')}</DialogTitle>
            <DialogDescription>{t('accountPage.mfa.enrollDescription')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-3">
            {enrollment?.qrCodeDataUrl && (
              <img alt={t('accountPage.mfa.qrCodeAlt')} className="mx-auto size-48 rounded-md border border-border bg-white p-2" src={enrollment.qrCodeDataUrl} />
            )}
            <Field hint={t('accountPage.mfa.uriHint')} label={t('accountPage.mfa.uri')}>
              <Input className="font-mono text-xs" readOnly value={enrollment?.otpauthUrl ?? ''} />
            </Field>
            <Field hint={t('accountPage.mfa.secretHint')} label={t('accountPage.mfa.secret')}>
              <Input className="font-mono" readOnly value={enrollment?.secret ?? ''} />
            </Field>
            <Field label={t('accountPage.mfa.confirmCode')} required>
              <Input
                autoComplete="one-time-code"
                inputMode="numeric"
                placeholder={t('accountPage.mfa.otpPlaceholder')}
                value={confirmationCode}
                onChange={event => setConfirmationCode(event.target.value)}
              />
            </Field>
          </div>
          <DialogFooter>
            <Button disabled={confirm.isPending} variant="secondary" onClick={() => setEnrollment(undefined)}>{t('cancel')}</Button>
            <Button disabled={confirm.isPending || confirmationCode.trim().length < 6} onClick={() => confirm.mutate({ code: confirmationCode.trim() })}>
              {t('accountPage.mfa.confirmEnable')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={recoveryCodes.length > 0} onOpenChange={open => !open && setRecoveryCodes([])}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('accountPage.mfa.recoveryTitle')}</DialogTitle>
            <DialogDescription>{t('accountPage.mfa.recoveryDescription')}</DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-2 rounded-md border border-border bg-muted/40 p-3 font-mono text-sm">
            {recoveryCodes.map(code => <code key={code}>{code}</code>)}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={copyRecoveryCodes}>
              <Copy size={16} />
              {t('accountPage.mfa.copyRecovery')}
            </Button>
            <Button onClick={() => setRecoveryCodes([])}>{t('accountPage.mfa.recoveryAcknowledged')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
