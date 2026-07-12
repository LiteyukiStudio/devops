import type { DataExportAuthorization } from '@/api'
import { api, deploymentTargetDataExportUrl } from '@/api'

interface ExportWindow {
  close: () => void
  location: Pick<Location, 'replace'>
  opener: unknown
}

interface DataExportDependencies {
  authorize?: (projectId: string, applicationId: string, targetId: string) => Promise<DataExportAuthorization>
  openWindow?: () => ExportWindow | null
}

export async function openDeploymentTargetDataExport(
  projectId: string,
  applicationId: string,
  targetId: string,
  dependencies: DataExportDependencies = {},
) {
  const exportWindow = (dependencies.openWindow ?? (() => window.open('about:blank', '_blank')))()
  if (!exportWindow)
    throw new Error('data_export_window_blocked')

  exportWindow.opener = null
  try {
    const authorization = await (dependencies.authorize ?? api.authorizeDeploymentTargetDataExport)(projectId, applicationId, targetId)
    const baseExportUrl = deploymentTargetDataExportUrl(projectId, applicationId, targetId)
    const exportUrl = new URL(baseExportUrl, window.location.origin)
    exportUrl.searchParams.set('ticket', authorization.ticket)
    exportWindow.location.replace(
      baseExportUrl.startsWith('http://') || baseExportUrl.startsWith('https://')
        ? exportUrl.toString()
        : `${exportUrl.pathname}${exportUrl.search}`,
    )
  }
  catch (error) {
    exportWindow.close()
    throw error
  }
}
