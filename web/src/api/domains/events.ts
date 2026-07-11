import type { PaginatedResponse, PlatformEvent, PlatformEventCatalogEntry, PlatformEventListParams } from '../types'
import { paginationQuery, request } from '../core'

function platformEventQuery(params: PlatformEventListParams) {
  const search = new URLSearchParams(paginationQuery(params))
  const filters: Array<[string, string | undefined]> = [
    ['projectId', params.projectId],
    ['applicationId', params.applicationId],
    ['deploymentTargetId', params.deploymentTargetId],
    ['category', params.category],
    ['type', params.type],
    ['severity', params.severity],
    ['status', params.status],
    ['dateFrom', params.dateFrom],
    ['dateTo', params.dateTo],
  ]
  for (const [key, value] of filters) {
    if (value)
      search.set(key, value)
  }
  return search.toString()
}

export const eventsApi = {
  listPlatformEvents: (params: PlatformEventListParams) =>
    request<PaginatedResponse<PlatformEvent>>(`/events?${platformEventQuery(params)}`),
  getPlatformEvent: (eventId: string) =>
    request<PlatformEvent>(`/events/${encodeURIComponent(eventId)}`),
  listPlatformEventCatalog: () =>
    request<PlatformEventCatalogEntry[]>('/events/catalog'),
}
