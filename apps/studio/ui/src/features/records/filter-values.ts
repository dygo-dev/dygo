import type { MetadataField } from '../metadata/metadata.api'
import type { RecordListFilter } from './query'
import { selectOptions } from '../../renderers/records/record-field-utils.ts'

export function filterInputType(field: MetadataField): string {
  if (['int', 'bigint', 'decimal', 'currency', 'float'].includes(field.type)) return 'number'
  if (field.type === 'datetime') return 'datetime-local'
  return ['date', 'time'].includes(field.type) ? field.type : 'text'
}

export function filterDisplayValue(value: string, type: string): string {
  if (type !== 'datetime' || !value || Number.isNaN(Date.parse(value))) return value
  const date = new Date(value)
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 19)
}

export function filterWireValue(value: string, type: string): string {
  return type === 'datetime' && value && !Number.isNaN(Date.parse(value)) ? new Date(value).toISOString() : value
}

// Validate the complete saved query before applying it; route normalization may discard invalid terms.
export function validateFilters(filters: RecordListFilter[], fields: MetadataField[]): string {
  for (const filter of filters) {
    if (filter.field === 'name' && filter.operator === 'contains' && filter.value?.trim()) continue
    const field = fields.find((item) => item.name === filter.field && item.listable && !item['write-only'])
    const operator = field?.filter?.operators.find((item) => item.key === filter.operator)
    if (!field || !operator) return `Filter ${filter.field} is no longer available.`
    if (operator.arity === 'none') continue
    const values = operator.arity === 'range' ? (filter.value ?? '').split('..') : [filter.value ?? '']
    if ((operator.arity === 'range' && values.length !== 2) || values.some((value) => !value.trim())) return `Enter a value for ${field.label || field.name}.`
    for (const value of values) {
      if (field.type === 'select' && !selectOptions(field).some((option) => option.value === value)) return `Choose an available ${field.label} value.`
      if (field.type === 'boolean' && !['true', 'false'].includes(value)) return `Choose true or false for ${field.label}.`
      if (filterInputType(field) === 'number' && !/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(value)) return `Enter a number for ${field.label}.`
      if (['int', 'bigint'].includes(field.type) && !/^[+-]?\d+$/.test(value)) return `Enter a whole number for ${field.label}.`
      if (['date', 'datetime'].includes(field.type) && Number.isNaN(Date.parse(value))) return `Enter a valid ${field.type} for ${field.label}.`
    }
  }
  return ''
}
