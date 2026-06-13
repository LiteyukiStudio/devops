export interface RuntimeDataVolumeRow {
  id: string
  name: string
  mountPath: string
  capacity: string
}

export function defaultRuntimeDataVolumeRow(): RuntimeDataVolumeRow {
  return { id: crypto.randomUUID(), capacity: '1Gi', mountPath: '/data', name: 'data' }
}

export function emptyRuntimeDataVolumeRow(index: number): RuntimeDataVolumeRow {
  return { id: crypto.randomUUID(), capacity: '1Gi', mountPath: '', name: `data-${index + 1}` }
}

export function parseRuntimeDataVolumes(value?: string, fallbackMountPath = '/data', fallbackCapacity = '1Gi'): RuntimeDataVolumeRow[] {
  const trimmed = value?.trim() ?? ''
  if (!trimmed || trimmed === '[]') {
    return [{
      id: crypto.randomUUID(),
      capacity: fallbackCapacity || '1Gi',
      mountPath: fallbackMountPath || '/data',
      name: 'data',
    }]
  }
  try {
    const parsed = JSON.parse(trimmed)
    if (Array.isArray(parsed)) {
      const rows = parsed.map((item, index) => ({
        id: crypto.randomUUID(),
        capacity: String(item?.capacity ?? '1Gi'),
        mountPath: String(item?.mountPath ?? ''),
        name: String(item?.name ?? `data-${index + 1}`),
      })).filter(row => row.name.trim() || row.mountPath.trim() || row.capacity.trim())
      return rows.length > 0 ? rows : [defaultRuntimeDataVolumeRow()]
    }
  }
  catch {
    return [defaultRuntimeDataVolumeRow()]
  }
  return [defaultRuntimeDataVolumeRow()]
}

export function serializeRuntimeDataVolumes(rows: RuntimeDataVolumeRow[]) {
  const volumes = rows
    .map(row => ({
      capacity: row.capacity.trim() || '1Gi',
      mountPath: row.mountPath.trim(),
      name: row.name.trim(),
    }))
    .filter(row => row.name || row.mountPath)
  return volumes.length > 0 ? JSON.stringify(volumes) : ''
}
