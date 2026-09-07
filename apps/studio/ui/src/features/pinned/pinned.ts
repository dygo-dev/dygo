export type PinnedItem = {
  type: 'entity' | 'page' | 'record'
  app: string
  entity?: string
  page?: string
  record?: string
  label: string
  path: string
}

export function pinnedItemID(item: PinnedItem): string {
  if (item.type === 'page') return `page:${item.app}:${item.page}`
  if (item.type === 'record') return `record:${item.app}:${item.entity}:${item.record}`
  return `entity:${item.app}:${item.entity}`
}

export function normalizePinnedItems(value: unknown): PinnedItem[] {
  if (!Array.isArray(value)) return []

  const seen = new Set<string>()
  return value.flatMap((candidate): PinnedItem[] => {
    if (!candidate || typeof candidate !== 'object') return []
    const item = candidate as Record<string, unknown>
    if (item.type !== 'entity' && item.type !== 'page' && item.type !== 'record') return []
    if (typeof item.app !== 'string' || typeof item.label !== 'string' || typeof item.path !== 'string') return []

    const normalized: PinnedItem = {
      type: item.type,
      app: item.app,
      label: item.label,
      path: item.path,
      ...(typeof item.entity === 'string' ? { entity: item.entity } : {}),
      ...(typeof item.page === 'string' ? { page: item.page } : {}),
      ...(typeof item.record === 'string' ? { record: item.record } : {}),
    }
    if ((normalized.type === 'page' && !normalized.page)
      || (normalized.type !== 'page' && !normalized.entity)
      || (normalized.type === 'record' && !normalized.record)) return []

    const id = pinnedItemID(normalized)
    if (seen.has(id)) return []
    seen.add(id)
    return [normalized]
  })
}
