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
  const multiFilters: Array<[string, string[] | undefined]> = [
    ['projectIds', params.projectIds],
    ['applicationIds', params.applicationIds],
    ['deploymentTargetIds', params.deploymentTargetIds],
    ['categories', params.categories],
    ['types', params.types],
    ['severities', params.severities],
    ['statuses', params.statuses],
  ]
  for (const [key, values] of multiFilters) {
    for (const value of values ?? []) {
      if (value)
        search.append(key, value)
    }
  }
  return search.toString()
}

export const eventsApi = {
  listPlatformEvents: (params: PlatformEventListParams) =>
    request<PaginatedResponse<PlatformEvent>>(`/events?${platformEventQuery(params)}`)
      .then(response => ({
        ...response,
        items: response.items.map(normalizePlatformEvent),
      })),
  getPlatformEvent: (eventId: string) =>
    request<PlatformEvent>(`/events/${encodeURIComponent(eventId)}`)
      .then(normalizePlatformEvent),
  listPlatformEventCatalog: () =>
    request<PlatformEventCatalogEntry[]>('/events/catalog'),
}

function normalizePlatformEvent(event: PlatformEvent): PlatformEvent {
  return {
    ...event,
    detail: isRecord(event.detail) ? event.detail : {},
    links: stringRecord(event.links),
    deliveryCount: Number.isFinite(event.deliveryCount) ? event.deliveryCount : 0,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function stringRecord(value: unknown) {
  if (!isRecord(value))
    return {}
  return Object.fromEntries(Object.entries(value).filter((entry): entry is [string, string] => typeof entry[1] === 'string'))
}
