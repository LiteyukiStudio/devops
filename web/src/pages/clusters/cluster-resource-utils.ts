import type { ClusterResource, CurrentUser } from '@/api'

export function canDeleteClusterResource(user: CurrentUser | undefined, item: ClusterResource) {
  return user?.role === 'platform_admin' || Boolean(item.projectId?.trim())
}
