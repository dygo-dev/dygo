import type { MetadataLinkFilter } from '@/features/metadata/metadata.api'
import type { RecordData } from '@/features/records/records.api'

export function resolveLinkFilterValue(value: unknown, currentValues: RecordData): unknown {
  if (typeof value !== 'string') {
    return value
  }
  const token = value.match(/^\$(?:\{([^}]+)\}|([a-z][a-z0-9-]*))$/i)
  if (!token) {
    return value
  }
  return currentValues[token[1] || token[2] || '']
}

export function resolveLinkFilters(filters: MetadataLinkFilter[], currentValues: RecordData) {
  return filters.flatMap((filter) => {
    const value = resolveLinkFilterValue(filter.value, currentValues)
    if (filter.operator !== 'empty' && filter.operator !== 'not-empty' && (value === undefined || value === null || value === '')) {
      return []
    }
    return [{
      field: filter.field,
      operator: filter.operator,
      ...(filter.operator === 'empty' || filter.operator === 'not-empty' ? {} : { value: String(value) }),
    }]
  })
}
