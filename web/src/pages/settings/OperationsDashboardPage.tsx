import { useQuery } from '@tanstack/react-query'
import { Settings } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { api } from '@/api'
import { useSession } from '@/app/session-context'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ForbiddenPage } from '@/components/common/forbidden-page'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

const OPERATIONS_DASHBOARD_URL_KEY = 'site.operationsDashboardUrl'

export function OperationsDashboardPage() {
  const { t } = useTranslation()
  const { user } = useSession()
  const isPlatformAdmin = user?.role === 'platform_admin'
  const configs = useQuery({
    queryKey: ['configs'],
    queryFn: api.getConfigs,
    enabled: isPlatformAdmin,
  })
  const dashboardUrl = configs.data?.[OPERATIONS_DASHBOARD_URL_KEY]?.trim() ?? ''
  const iframeUrl = useMemo(() => resolveIframeUrl(dashboardUrl), [dashboardUrl])

  if (!isPlatformAdmin)
    return <ForbiddenPage />

  if (configs.isLoading)
    return <OperationsDashboardSkeleton />

  if (configs.isError) {
    return (
      <ErrorState
        description={t('operationsDashboardPage.loadFailedDescription')}
        title={t('operationsDashboardPage.loadFailedTitle')}
      />
    )
  }

  if (!dashboardUrl) {
    return (
      <EmptyState
        actions={(
          <Button asChild>
            <Link to="/settings/site">
              <Settings size={16} />
              {t('operationsDashboardPage.configure')}
            </Link>
          </Button>
        )}
        description={t('operationsDashboardPage.emptyDescription')}
        title={t('operationsDashboardPage.emptyTitle')}
      />
    )
  }

  if (!iframeUrl) {
    return (
      <ErrorState
        description={t('operationsDashboardPage.invalidDescription')}
        title={t('operationsDashboardPage.invalidTitle')}
      />
    )
  }

  return (
    <Card className="overflow-hidden p-0">
      <iframe
        className="h-[72dvh] min-h-128 max-h-192 w-full border-0 bg-background"
        referrerPolicy="strict-origin-when-cross-origin"
        src={iframeUrl}
        title={t('operationsDashboard')}
      />
    </Card>
  )
}

function OperationsDashboardSkeleton() {
  return (
    <Card className="overflow-hidden p-0">
      <div className="h-[72dvh] min-h-128 max-h-192 w-full animate-pulse bg-muted" />
    </Card>
  )
}

function resolveIframeUrl(rawUrl: string) {
  try {
    const parsed = new URL(rawUrl)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:')
      return ''
    return parsed.toString()
  }
  catch {
    return ''
  }
}
