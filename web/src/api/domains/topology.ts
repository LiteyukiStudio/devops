import type {
  ProjectTopology,
  ProjectTopologyListParams,
  ProjectTopologyManualEdge,
  ProjectTopologyManualEdgePage,
  ProjectTopologyManualEdgePayload,
  ProjectTopologyQuery,
  ServiceBindingCheckResult,
  ServiceBindingMutationResult,
  ServiceBindingPage,
  ServiceBindingPayload,
} from '../topology-types'
import { paginationQuery, request } from '../core'

function topologyQuery(params?: ProjectTopologyQuery) {
  const search = new URLSearchParams()
  if (params?.stage)
    search.set('stage', params.stage)
  if (params?.applicationId)
    search.set('applicationId', params.applicationId)
  if (params?.origins?.length)
    search.set('origins', params.origins.join(','))
  const query = search.toString()
  return query ? `?${query}` : ''
}

function topologyListQuery(params: ProjectTopologyListParams) {
  const search = new URLSearchParams(paginationQuery(params))
  if (params.enabled !== undefined)
    search.set('enabled', String(params.enabled))
  return search.toString()
}

export const topologyApi = {
  getProjectTopology: (projectId: string, params?: ProjectTopologyQuery) =>
    request<ProjectTopology>(`/projects/${projectId}/topology${topologyQuery(params)}`),
  listServiceBindings: (projectId: string, params: ProjectTopologyListParams) =>
    request<ServiceBindingPage>(`/projects/${projectId}/service-bindings?${topologyListQuery(params)}`),
  createServiceBinding: (projectId: string, payload: ServiceBindingPayload) =>
    request<ServiceBindingMutationResult>(`/projects/${projectId}/service-bindings`, { method: 'POST', body: JSON.stringify(payload) }),
  updateServiceBinding: (projectId: string, bindingId: string, payload: ServiceBindingPayload) =>
    request<ServiceBindingMutationResult>(`/projects/${projectId}/service-bindings/${bindingId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteServiceBinding: (projectId: string, bindingId: string) =>
    request<ServiceBindingMutationResult>(`/projects/${projectId}/service-bindings/${bindingId}`, { method: 'DELETE' }),
  checkServiceBinding: (projectId: string, bindingId: string) =>
    request<ServiceBindingCheckResult>(`/projects/${projectId}/service-bindings/${bindingId}/check`, { method: 'POST' }),
  listProjectTopologyEdges: (projectId: string, params: ProjectTopologyListParams) =>
    request<ProjectTopologyManualEdgePage>(`/projects/${projectId}/topology-edges?${topologyListQuery(params)}`),
  createProjectTopologyEdge: (projectId: string, payload: ProjectTopologyManualEdgePayload) =>
    request<ProjectTopologyManualEdge>(`/projects/${projectId}/topology-edges`, { method: 'POST', body: JSON.stringify(payload) }),
  updateProjectTopologyEdge: (projectId: string, edgeId: string, payload: ProjectTopologyManualEdgePayload) =>
    request<ProjectTopologyManualEdge>(`/projects/${projectId}/topology-edges/${edgeId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteProjectTopologyEdge: (projectId: string, edgeId: string) =>
    request<void>(`/projects/${projectId}/topology-edges/${edgeId}`, { method: 'DELETE' }),
}
