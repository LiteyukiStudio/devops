import type { UseFormReturn } from 'react-hook-form'
import type { ReleaseForm } from './application-deployments-panel-utils'
import type { BuildRun, DeploymentTarget } from '@/api'
import { useTranslation } from 'react-i18next'
import { CopyableHoverText } from '@/components/common/copyable-hover-text'
import { buildRunImageRef, buildRunOptionLabel } from '@/components/common/deployment-build-runs'
import { FormField as Field } from '@/components/common/form-field'
import { ProgressiveSection } from '@/components/common/progressive-section'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'
import { shortImageRef } from './application-deployments-panel-utils'

interface ApplicationCreateReleaseDialogProps {
  form: UseFormReturn<ReleaseForm>
  open: boolean
  pending: boolean
  releaseReadyTargets: DeploymentTarget[]
  selectableBuildRuns: BuildRun[]
  selectedTarget?: DeploymentTarget
  onOpenChange: (open: boolean) => void
  onSubmit: (values: ReleaseForm) => void
}

export function ApplicationCreateReleaseDialog({
  form,
  onOpenChange,
  onSubmit,
  open,
  pending,
  releaseReadyTargets,
  selectableBuildRuns,
  selectedTarget,
}: ApplicationCreateReleaseDialogProps) {
  const { t } = useTranslation()
  const imageRef = form.watch('imageRef')
  const selectedBuildRun = selectableBuildRuns.find(run => run.id === form.watch('buildRunId'))
  const imageSummary = imageRef || (selectedBuildRun ? buildRunImageRef(selectedBuildRun) : '')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{t('deploymentsPage.createRelease')}</DialogTitle>
          <DialogDescription>{t('deploymentsPage.releaseDialogDescription')}</DialogDescription>
        </DialogHeader>
        <form className="grid gap-3" onSubmit={form.handleSubmit(onSubmit)}>
          <input type="hidden" {...form.register('imageRef', { required: true })} />
          {selectedTarget?.sourceType !== 'image' && (
            <Field hint={t('deploymentsPage.buildRunHint')} label={t('deploymentsPage.buildRun')} required>
              <Select {...form.register('buildRunId', { required: true })}>
                <option value="">{t('common.select')}</option>
                {selectableBuildRuns.map(run => <option key={run.id} value={run.id}>{buildRunOptionLabel(run)}</option>)}
              </Select>
            </Field>
          )}
          <Field hint={selectedTarget ? t('deploymentsPage.releaseTargetLockedHint') : undefined} label={t('buildsPage.buildConfig')} required>
            {selectedTarget
              ? (
                  <>
                    <input type="hidden" {...form.register('deploymentTargetId', { required: true })} />
                    <div className="rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">
                      <span className="font-medium text-foreground">{selectedTarget.name}</span>
                      <span className="ml-2 text-muted-foreground">{t(`deploymentsPage.stageLabels.${selectedTarget.stage}`, { defaultValue: selectedTarget.stage })}</span>
                    </div>
                  </>
                )
              : (
                  <Select {...form.register('deploymentTargetId', { required: true })}>
                    <option value="">{t('common.select')}</option>
                    {releaseReadyTargets.map(target => <option key={target.id} value={target.id}>{target.name}</option>)}
                  </Select>
                )}
          </Field>
          <Field hint={t('deploymentsPage.releaseImageSummaryHint')} label={t('deploymentsPage.imageSummary')} required>
            <div className="rounded-md border border-border bg-muted/40 px-3 py-2">
              <CopyableHoverText
                className="max-w-full font-mono text-sm"
                display={imageSummary ? shortImageRef(imageSummary) : t('common.select')}
                value={imageSummary}
              />
            </div>
          </Field>
          <ProgressiveSection
            description={t('deploymentsPage.releaseImageOverrideDescription')}
            summary={imageSummary ? shortImageRef(imageSummary) : t('common.select')}
            title={t('deploymentsPage.releaseImageOverride')}
          >
            <Field hint={t('deploymentsPage.releaseImageOverrideHint')} label={t('deploymentsPage.image')} required>
              <Input
                value={imageRef}
                onChange={event => form.setValue('imageRef', event.target.value, { shouldDirty: true, shouldValidate: true })}
              />
            </Field>
          </ProgressiveSection>
          <DialogFooter>
            <Button disabled={!form.formState.isValid || pending} type="submit">{t('common.save')}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
