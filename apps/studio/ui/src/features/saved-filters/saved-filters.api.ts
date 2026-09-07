import { apiRequest, type DataEnvelope } from '@/features/api/client'
import { RecordApiError } from '@/features/records/records.api'
import type { RecordListFilter } from '@/features/records/query'

export type SavedFilter = { id: number; entity: string; label: string; filters: RecordListFilter[]; validationError?: string }
export async function savedFilterRequest<T>(path: string, method = 'GET', body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await apiRequest<DataEnvelope<T>, RecordApiError>(`/api/v1/studio/saved-filters${path}`, {
    method, signal, ...(body === undefined ? {} : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  }, {
    error: RecordApiError, fallbackCode: 'saved_filter_failed', invalidResponseMessage: 'Could not read saved filters.',
    message: (payload) => payload.error?.message ?? 'Could not save filter changes.',
  })
  return response.data
}
