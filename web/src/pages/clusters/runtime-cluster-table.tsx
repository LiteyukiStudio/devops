import type { CurrentUser, Project, RuntimeCluster } from '@/api'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataList } from '@/components/common/data-list'
import { EditActionButton } from '@/components/common/edit-action-button'
import { StatusValueBadge } from '@/components/common/status-badge'
import { Button } from '@/components/ui/button'
import { canManageCluster, clusterTypeLabel, gatewayDomainSuffixSummary, gatewayPublicPortSummary, scopeLabel } from './cluster-helpers'

export function RuntimeClusterTable({ clusters, projects, user, onDelete, onEdit, onTest }: {
  clusters: RuntimeCluster[]
  projects: Project[]
  user?: CurrentUser
  onDelete: (cluster: RuntimeCluster) => void
  onEdit: (cluster: RuntimeCluster) => void
  onTest: (clusterId: string) => void
}) {
  const { t } = useTranslation()
  const projectMap = Object.fromEntries(projects.map(project => [project.id, project]))

  return (
    <DataList
      columns={[
        { key: 'name', header: t('common.name'), width: 'primary', render: item => item.name },
        { key: 'type', header: t('common.type'), width: 'secondary', render: item => clusterTypeLabel(item.type, t) },
        { key: 'scope', header: t('common.scope'), width: 'status', render: item => scopeLabel(item, projectMap, t) },
        { key: 'default', header: t('clustersPage.defaultCluster'), width: 'status', render: item => item.isDefault ? t('common.yes') : t('common.no') },
        { key: 'buildConcurrency', header: t('clustersPage.maxConcurrentBuilds'), width: 'number', render: item => item.maxConcurrentBuilds || 4 },
        { key: 'gatewayRootDomain', header: t('clustersPage.gatewayDomainSuffixes'), width: 'secondary', render: item => gatewayDomainSuffixSummary(item) },
        { key: 'gatewayPublicScheme', header: t('clustersPage.gatewayPublicScheme'), width: 'compact', render: item => item.gatewayPublicScheme || 'http' },
        { key: 'gatewayPublicPort', header: t('clustersPage.gatewayPublicPort'), width: 'compact', render: item => gatewayPublicPortSummary(item) },
        { key: 'status', header: t('common.status'), width: 'status', render: item => <StatusValueBadge value={item.status} /> },
        {
          key: 'actions',
          header: t('common.actions'),
          className: 'text-right whitespace-nowrap',
          sticky: 'right',
          width: 'actions',
          render: item => canManageCluster(item, user?.id, user?.role)
            ? (
                <div className="flex justify-end gap-2">
                  <Button size="sm" variant="ghost" onClick={() => onTest(item.id)}>{t('common.test')}</Button>
                  <EditActionButton label={t('common.edit')} onClick={() => onEdit(item)} />
                  <Button size="sm" variant="ghost" onClick={() => onDelete(item)}>
                    <Trash2 className="size-4" />
                    {t('common.delete')}
                  </Button>
                </div>
              )
            : <span className="text-xs text-muted-foreground">{t('common.viewOnly')}</span>,
        },
      ]}
      emptyTitle={t('deploymentsPage.emptyClusters')}
      items={clusters}
      rowKey={item => item.id}
    />
  )
}
