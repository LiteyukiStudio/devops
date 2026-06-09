import type { UseFormRegisterReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FormField as Field } from '@/components/common/form-field'
import { Input } from '@/components/ui/input'
import { NativeSelect as Select } from '@/components/ui/native-select'

export function GatewayRouteFormFields({
  applicationIdField,
  applications = [],
  environmentIdField,
  environments,
  hostField,
  pathField,
  servicePortField,
  showApplication = false,
  stageField,
  tlsModeField,
}: {
  applicationIdField?: UseFormRegisterReturn<'applicationId'>
  applications?: Array<{ id: string, name: string }>
  environmentIdField: UseFormRegisterReturn<'environmentId'>
  environments: Array<{ id: string, name: string }>
  hostField: UseFormRegisterReturn<'host'>
  pathField: UseFormRegisterReturn<'path'>
  servicePortField: UseFormRegisterReturn<'servicePort'>
  showApplication?: boolean
  stageField: UseFormRegisterReturn<'stage'>
  tlsModeField: UseFormRegisterReturn<'tlsMode'>
}) {
  const { t } = useTranslation()

  return (
    <>
      {showApplication && (
        <Field label={t('apps.title')} required>
          <Select {...applicationIdField}>
            <option value="">{t('common.select')}</option>
            {applications.map(app => <option key={app.id} value={app.id}>{app.name}</option>)}
          </Select>
        </Field>
      )}
      <Field label={t('deploymentsPage.environment')}>
        <Select {...environmentIdField}>
          <option value="">{t('common.none')}</option>
          {environments.map(environment => <option key={environment.id} value={environment.id}>{environment.name}</option>)}
        </Select>
      </Field>
      <Field hint={t('gatewayRoutesPage.hostHint')} label={t('gatewayRoutesPage.host')} required={!showApplication}>
        <Input {...hostField} />
      </Field>
      <Field label={t('deploymentsPage.stage')}>
        <Select {...stageField}>
          <option value="dev">{t('deploymentsPage.stageDev')}</option>
          <option value="test">{t('deploymentsPage.stageTest')}</option>
          <option value="staging">{t('deploymentsPage.stageStaging')}</option>
          <option value="prod">{t('deploymentsPage.stageProd')}</option>
        </Select>
      </Field>
      <Field label={t('gatewayRoutesPage.path')}>
        <Input {...pathField} />
      </Field>
      <Field label={t('gatewayRoutesPage.servicePort')}>
        <Input {...servicePortField} type="number" />
      </Field>
      <Field label={t('gatewayRoutesPage.tlsMode')}>
        <Select {...tlsModeField}>
          <option value="http-only">{t('gatewayRoutesPage.tlsHttpOnly')}</option>
          <option value="http-challenge">{t('gatewayRoutesPage.tlsHttpChallenge')}</option>
          <option value="manual-cert">{t('gatewayRoutesPage.tlsManualCert')}</option>
        </Select>
      </Field>
    </>
  )
}
