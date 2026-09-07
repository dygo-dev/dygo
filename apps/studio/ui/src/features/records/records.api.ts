import { ApiClientError, apiRequest, type ApiErrorEnvelope, type DataEnvelope, type ListEnvelope } from '@/features/api/client'
import { buildRecordListQuery, type ListRecordsParams } from './query'

export type RecordValue = unknown

export type RecordData = Record<string, RecordValue>

export type RecordListMeta = {
  limit: number
  offset: number
  count: number
  total?: number
}

export type ActivityActor = {
  id: number
  email: string
  'full-name': string
}

export type ActivityEntry = {
  id: number
  'created-at': string
  entity: string
  'record-id': number
  kind: string
  operation: string
  status: string
  title: string
  message: string
  actor: ActivityActor | null
  changes: unknown
  snapshot: unknown
  details: unknown
}

export type ActivityListMeta = {
  limit: number
  offset: number
  count: number
}

export type ImportRowInfo = { 'row-number': number, status: string, error?: string, 'record-id'?: number }
export type ImportInfo = {
  id: number
  name: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  'total-rows': number
  'processed-rows': number
  'succeeded-rows': number
  'failed-rows': number
  rows?: ImportRowInfo[]
}
export type FileInfo = { id: number, filename: string }

type ListRecordsOptions = {
  signal?: AbortSignal
}

type ReadRecordOptions = {
  signal?: AbortSignal
}

type ActivityOptions = {
  signal?: AbortSignal
}

export class RecordApiError extends ApiClientError {
  constructor(code: string, message: string, details?: Record<string, unknown>) {
    super('RecordApiError', code, message, details)
  }
}

export async function listRecords(entity: string, params: ListRecordsParams, options: ListRecordsOptions = {}): Promise<ListEnvelope<RecordData[], RecordListMeta>> {
  const query = buildRecordListQuery(params)

  return apiRequest<ListEnvelope<RecordData[], RecordListMeta>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}?${query.toString()}`, {
    method: 'GET',
    signal: options.signal,
  }, recordRequestOptions('records_failed'))
}

export async function exportRecordsCSV(entity: string, params: ListRecordsParams): Promise<Blob> {
  const query = buildRecordListQuery(params)
  const response = await fetch(`/api/v1/records/${encodeURIComponent(entity)}/export?${query.toString()}`, { credentials: 'include' })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({})) as ApiErrorEnvelope
    throw new RecordApiError(payload.error?.code ?? 'export_failed', recordErrorMessage(payload), payload.error?.details)
  }
  return response.blob()
}

export async function startCSVImport(app: string, entity: string, file: Blob): Promise<ImportInfo> {
  const form = new FormData()
  form.append('app', app)
  form.append('entity', entity)
  form.append('file', file, 'import.csv')
  const payload = await apiRequest<DataEnvelope<ImportInfo>, RecordApiError>('/api/v1/imports', { method: 'POST', body: form }, recordRequestOptions('import_failed'))
  return payload.data
}

export async function getImportStatus(id: number): Promise<ImportInfo> {
  const payload = await apiRequest<DataEnvelope<ImportInfo>, RecordApiError>(`/api/v1/imports/${id}`, { method: 'GET' }, recordRequestOptions('import_status_failed'))
  return payload.data
}

export async function uploadRecordFile(app: string, entity: string, recordID: number, field: string, file: File): Promise<FileInfo> {
  const form = new FormData()
  form.append('app', app)
  form.append('entity', entity)
  form.append('record-id', String(recordID))
  form.append('field', field)
  form.append('file', file)
  const payload = await apiRequest<DataEnvelope<FileInfo>, RecordApiError>('/api/v1/files', { method: 'POST', body: form }, recordRequestOptions('file_upload_failed'))
  return payload.data
}

export async function getRecordByName(entity: string, recordName: string, options: ReadRecordOptions = {}): Promise<RecordData> {
  const payload = await apiRequest<DataEnvelope<RecordData>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/name/${encodeURIComponent(recordName)}`, {
    method: 'GET',
    signal: options.signal,
  }, recordRequestOptions('record_lookup_failed'))

  return payload.data
}

export async function getSingleRecord(entity: string, options: ReadRecordOptions = {}): Promise<RecordData> {
  const payload = await apiRequest<DataEnvelope<RecordData>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/single`, {
    method: 'GET',
    signal: options.signal,
  }, recordRequestOptions('single_record_lookup_failed'))

  return payload.data
}

export async function createRecord(entity: string, data: RecordData): Promise<RecordData> {
  const payload = await apiRequest<DataEnvelope<RecordData>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data }),
  }, recordRequestOptions('record_create_failed'))

  return payload.data
}

export async function updateRecord(entity: string, id: string | number, data: RecordData): Promise<RecordData> {
  const payload = await apiRequest<DataEnvelope<RecordData>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/${encodeURIComponent(String(id))}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data }),
  }, recordRequestOptions('record_update_failed'))

  return payload.data
}

export async function deleteRecord(entity: string, id: string | number): Promise<void> {
  await apiRequest<DataEnvelope<{ deleted: boolean }>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
  }, recordRequestOptions('record_delete_failed'))
}

export async function updateSingleRecord(entity: string, data: RecordData): Promise<RecordData> {
  const payload = await apiRequest<DataEnvelope<RecordData>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/single`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data }),
  }, recordRequestOptions('single_record_update_failed'))

  return payload.data
}

export async function listRecordActivity(entity: string, id: string | number, options: ActivityOptions = {}): Promise<ListEnvelope<ActivityEntry[], ActivityListMeta>> {
  const query = new URLSearchParams({ limit: '50', offset: '0' })
  return apiRequest<ListEnvelope<ActivityEntry[], ActivityListMeta>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/${encodeURIComponent(String(id))}/activity?${query.toString()}`, {
    method: 'GET',
    signal: options.signal,
  }, recordRequestOptions('activity_failed'))
}

export async function addRecordComment(entity: string, id: string | number, message: string): Promise<{ created: boolean }> {
  const payload = await apiRequest<DataEnvelope<{ created: boolean }>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/${encodeURIComponent(String(id))}/activity`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
  }, recordRequestOptions('comment_failed'))

  return payload.data
}

export async function executeRecordAction(entity: string, action: string, records: number[], input: Record<string, unknown> = {}): Promise<unknown> {
  const payload = await apiRequest<DataEnvelope<unknown>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/actions/${encodeURIComponent(action)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ records, input }),
  }, recordRequestOptions('action_failed'))

  return payload.data
}

function recordRequestOptions(fallbackCode: string) {
  return {
    error: RecordApiError,
    fallbackCode,
    invalidResponseMessage: 'Studio could not read the records response.',
    message: recordErrorMessage,
  }
}

function recordErrorMessage(payload: ApiErrorEnvelope): string {
  switch (payload.error?.code) {
    case 'unauthenticated':
      return 'Sign in to load records.'
    case 'forbidden':
      return 'You do not have permission to read these records.'
    case 'not_found':
      return payload.error.message ?? 'Studio could not find this record list.'
    case 'schema_not_ready':
      return 'Record metadata is not ready yet. Run dygo db migrate, then try again.'
    default:
      return payload.error?.message ?? 'Studio could not load records.'
  }
}

export type SecretStatus = { fields: Record<string, boolean>, collections?: Record<string, Record<string, Record<string, boolean>>> }
export async function getSecretStatus(entity: string, id: number, signal?: AbortSignal): Promise<SecretStatus> {
  const response = await apiRequest<DataEnvelope<SecretStatus>, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/${id}/secret-status`, { method: 'GET', signal }, recordRequestOptions('secret_status_failed'))
  return response.data
}
