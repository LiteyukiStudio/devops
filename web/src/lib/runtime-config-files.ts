export interface RuntimeConfigFileRow {
  id: string
  path: string
  content: string
}

export function emptyRuntimeConfigFileRow(): RuntimeConfigFileRow {
  return { id: crypto.randomUUID(), content: '', path: '' }
}

export function parseRuntimeConfigFiles(value: string): RuntimeConfigFileRow[] {
  const trimmed = value.trim()
  if (!trimmed || trimmed === '[]')
    return []
  try {
    const parsed = JSON.parse(trimmed)
    if (Array.isArray(parsed)) {
      return parsed.map(item => ({
        id: crypto.randomUUID(),
        content: String(item?.content ?? ''),
        path: String(item?.path ?? ''),
      })).filter(row => row.path.trim() || row.content.trim())
    }
  }
  catch {
    return []
  }
  return []
}

export function serializeRuntimeConfigFiles(rows: RuntimeConfigFileRow[]) {
  const files = rows
    .map(row => ({ content: row.content, path: row.path.trim() }))
    .filter(row => row.path || row.content)
  return files.length > 0 ? JSON.stringify(files) : ''
}

export function runtimeConfigFileCount(value: string) {
  return parseRuntimeConfigFiles(value).filter(row => row.path.trim()).length
}
