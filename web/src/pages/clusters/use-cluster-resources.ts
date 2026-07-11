import type { ClusterResourcePagination } from './cluster-resources-panel'
import type { ClusterResource, CurrentUser, RuntimeCluster } from '@/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/api'
import { canDeleteClusterResource } from './cluster-resource-utils'

const RESOURCE_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

export function useClusterResources({ activeTab, manageableClusters, user }: {
  activeTab: string
  manageableClusters: RuntimeCluster[]
  user?: CurrentUser
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedResourceClusterId, setSelectedResourceClusterId] = useState('')
  const [selectedResourceKeys, setSelectedResourceKeys] = useState<string[]>([])
  const [resourcePage, setResourcePage] = useState(1)
  const [resourcePageSize, setResourcePageSize] = useState(10)
  const [resourceToDelete, setResourceToDelete] = useState<ClusterResource | null>(null)
  const [resourcesToDelete, setResourcesToDelete] = useState<ClusterResource[]>([])
  const [eventResource, setEventResource] = useState<ClusterResource | null>(null)
  const [consoleResource, setConsoleResource] = useState<ClusterResource | null>(null)
  const [yamlResource, setYamlResource] = useState<ClusterResource | null>(null)

  const effectiveResourceClusterId = manageableClusters.some(cluster => cluster.id === selectedResourceClusterId)
    ? selectedResourceClusterId
    : manageableClusters[0]?.id ?? ''
  const selectedResourceCluster = manageableClusters.find(cluster => cluster.id === effectiveResourceClusterId)
  const resourceKind = activeTab === 'clusters' ? 'namespaces' : activeTab
  const clusterResources = useQuery({
    queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind, resourcePage, resourcePageSize],
    queryFn: () => api.listRuntimeClusterResourcesPage(selectedResourceCluster?.id ?? '', {
      kind: resourceKind,
      page: resourcePage,
      pageSize: resourcePageSize,
      sortBy: 'updatedAt',
      sortOrder: 'desc',
    }),
    enabled: activeTab !== 'clusters' && Boolean(selectedResourceCluster?.id),
  })
  const activeResourceItems = useMemo(
    () => activeTab === 'clusters' ? [] : clusterResources.data?.items ?? [],
    [activeTab, clusterResources.data?.items],
  )
  const activeResourceKeySet = useMemo(() => new Set(activeResourceItems.map(item => item.id)), [activeResourceItems])
  const visibleSelectedResourceKeys = useMemo(
    () => selectedResourceKeys.filter(key => activeResourceKeySet.has(key)),
    [activeResourceKeySet, selectedResourceKeys],
  )
  const selectedDeletableResources = useMemo(() => {
    const selectedKeys = new Set(visibleSelectedResourceKeys)
    return activeResourceItems.filter(item => selectedKeys.has(item.id) && canDeleteClusterResource(user, item))
  }, [activeResourceItems, user, visibleSelectedResourceKeys])
  const resourceEvents = useQuery({
    queryKey: ['runtime-cluster-resource-events', selectedResourceCluster?.id, eventResource?.kind, eventResource?.namespace, eventResource?.name],
    queryFn: () => api.listRuntimeClusterResourceEvents(selectedResourceCluster?.id ?? '', {
      kind: eventResource?.kind ?? '',
      namespace: eventResource?.namespace,
      name: eventResource?.name ?? '',
    }),
    enabled: Boolean(selectedResourceCluster?.id && eventResource),
  })
  const resourceYAML = useQuery({
    queryKey: ['runtime-cluster-resource-yaml', selectedResourceCluster?.id, yamlResource?.kind, yamlResource?.namespace, yamlResource?.name],
    queryFn: () => api.getRuntimeClusterResourceYAML(selectedResourceCluster?.id ?? '', {
      kind: yamlResource?.kind ?? '',
      namespace: yamlResource?.namespace,
      name: yamlResource?.name ?? '',
    }),
    enabled: Boolean(selectedResourceCluster?.id && yamlResource),
  })

  useEffect(() => {
    setResourcePage(1)
    setSelectedResourceKeys([])
  }, [activeTab, effectiveResourceClusterId])

  const deleteResource = useMutation({
    mutationFn: (resource: ClusterResource) => api.deleteRuntimeClusterResource(effectiveResourceClusterId, {
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
    }),
    onSuccess: () => {
      toast.success(t('clustersPage.resourceDeleted'))
      setResourceToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind] })
    },
    onError: error => toast.error(error.message),
  })
  const deleteResources = useMutation({
    mutationFn: async (resources: ClusterResource[]) => {
      for (const resource of resources) {
        await api.deleteRuntimeClusterResource(effectiveResourceClusterId, {
          kind: resource.kind,
          namespace: resource.namespace,
          name: resource.name,
        })
      }
    },
    onSuccess: (_, resources) => {
      toast.success(t('clustersPage.resourcesDeleted', { count: resources.length }))
      setResourcesToDelete([])
      setSelectedResourceKeys([])
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['runtime-cluster-resources', selectedResourceCluster?.id, resourceKind] })
    },
    onError: error => toast.error(error.message),
  })

  const resetPageAndSelection = () => {
    setResourcePage(1)
    setSelectedResourceKeys([])
  }
  const selectResourceCluster = (clusterId: string) => {
    setSelectedResourceClusterId(clusterId)
    resetPageAndSelection()
  }
  const resourcePagination: ClusterResourcePagination | undefined = activeTab === 'clusters'
    ? undefined
    : {
        page: clusterResources.data?.page ?? resourcePage,
        pageSize: clusterResources.data?.pageSize ?? resourcePageSize,
        pageInfoLabel: t('pagination.pageInfo', {
          page: clusterResources.data?.page ?? resourcePage,
          total: clusterResources.data?.total ?? 0,
          totalPages: clusterResources.data?.totalPages ?? 0,
        }),
        pageSizeOptions: RESOURCE_PAGE_SIZE_OPTIONS,
        total: clusterResources.data?.total ?? 0,
        totalPages: clusterResources.data?.totalPages ?? 0,
        onPageChange: (page) => {
          setResourcePage(page)
          setSelectedResourceKeys([])
        },
        onPageSizeChange: (pageSize) => {
          setResourcePageSize(pageSize)
          resetPageAndSelection()
        },
      }

  return {
    activeResourceItems,
    clusterResources,
    consoleResource,
    deleteResource,
    deleteResources,
    eventResource,
    resourceEvents,
    resourcePagination,
    resourceToDelete,
    resourceYAML,
    resourcesToDelete,
    selectedDeletableResources,
    selectedResourceCluster,
    selectedResourceKeys,
    visibleSelectedResourceKeys,
    yamlResource,
    resetPageAndSelection,
    selectResourceCluster,
    setConsoleResource,
    setEventResource,
    setResourceToDelete,
    setResourcesToDelete,
    setSelectedResourceKeys,
    setYamlResource,
  }
}
