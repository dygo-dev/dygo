// An empty editor preserves the saved value. Only the Clear control emits null.
export function secretSubmitValue(value: unknown, required: boolean, existing: boolean): { skip?: boolean, value?: unknown, error?: string } {
  if (value === null && !required) return { value: null }
  if (typeof value === 'string' && value !== '') return { value }
  if (required && (!existing || value === null)) return { skip: true, error: 'Enter a secret.' }
  return { skip: true }
}
