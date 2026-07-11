import type { BillingDeploymentSpend, Project } from '@/api'
import type { useBillingDisplay } from '@/lib/billing-display'
import { cn } from '@/lib/utils'

export type BillingDisplay = ReturnType<typeof useBillingDisplay>

export interface BillingListPage<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export function ProjectCell({
  fallbackName,
  fallbackSlug,
  project,
  unknownLabel,
}: {
  fallbackName?: string
  fallbackSlug?: string
  project?: Project
  unknownLabel: string
}) {
  const name = project?.name || fallbackName || unknownLabel
  const slug = project?.slug || fallbackSlug || '-'
  return (
    <span className="block min-w-0">
      <span className="block truncate font-medium">{name}</span>
      <span className="block truncate text-xs text-muted-foreground">{slug}</span>
    </span>
  )
}

type BillingApplicationRef = Pick<BillingDeploymentSpend, 'applicationName' | 'applicationSlug'> & {
  applicationId?: string
}

export function ApplicationCell({ item, unassignedLabel }: { item: BillingApplicationRef, unassignedLabel: string }) {
  return (
    <span className="block min-w-0">
      <span className="block truncate font-medium">{item.applicationName || unassignedLabel}</span>
      <span className="block truncate text-xs text-muted-foreground">{item.applicationSlug || '-'}</span>
    </span>
  )
}

type BillingDeploymentTargetRef = Pick<BillingDeploymentSpend, 'deploymentTargetName' | 'deploymentTargetStage'> & {
  deploymentTargetId?: string
}

export function DeploymentTargetCell({ item, unassignedLabel }: { item: BillingDeploymentTargetRef, unassignedLabel: string }) {
  return (
    <span className="block min-w-0">
      <span className="block truncate font-medium">{item.deploymentTargetName || unassignedLabel}</span>
      <span className="block truncate text-xs text-muted-foreground">{item.deploymentTargetStage || item.deploymentTargetId || '-'}</span>
    </span>
  )
}

export function SpendAmount({
  billingDisplay,
  strong = false,
  value,
}: {
  billingDisplay: BillingDisplay
  strong?: boolean
  value: string
}) {
  return (
    <span className={cn('tabular-nums text-foreground', strong && 'font-semibold')}>
      {billingDisplay.formatAmountWithUnit(value)}
    </span>
  )
}

export function ResourceCell({ resourceId, resourceType }: { resourceId: string, resourceType: string }) {
  return (
    <span className="block min-w-0">
      <span className="block truncate font-medium">{resourceType || '-'}</span>
      <span className="block truncate text-xs text-muted-foreground">{resourceId || '-'}</span>
    </span>
  )
}
