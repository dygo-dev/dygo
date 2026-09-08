import type { FieldOption, TextInputType } from '@/design'
import type { MetadataField } from '@/features/metadata/metadata.api'

export function textValue(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' || typeof value === 'bigint' || typeof value === 'boolean') {
    return String(value)
  }
  return JSON.stringify(value, null, 2)
}

export function booleanValue(value: unknown): boolean {
  return value === true
}

export function inputTypeForField(field: MetadataField): Exclude<TextInputType, 'password'> {
  switch (editorForField(field)) {
    case 'email':
      return 'email'
    case 'date':
      return 'date'
    case 'number':
      return 'number'
    default:
      return 'text'
  }
}

export function editorForField(field: MetadataField): string {
  return field.studio?.editor || field.type
}

export function isTextField(field: MetadataField): boolean {
  return ['text', 'email', 'number', 'link', 'date', 'datetime', 'time', 'password'].includes(editorForField(field))
}

export function isTextareaField(field: MetadataField): boolean {
  return editorForField(field) === 'textarea' || editorForField(field) === 'json'
}

export function selectOptions(field: MetadataField): FieldOption[] {
  const options = field.options
  if (!options || typeof options !== 'object') {
    return []
  }

  const values = Array.isArray(options) ? options : (options as { values?: unknown }).values
  if (!Array.isArray(values)) {
    return []
  }

  return values
    .filter((value): value is string | number => typeof value === 'string' || typeof value === 'number')
    .map((value) => ({ value: String(value), label: String(value) }))
}
