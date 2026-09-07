import { apiRequest } from '../api/client'
import { RecordApiError, type RecordData, type RecordListMeta } from './records.api'
import { buildRecordListQuery, type ListRecordsParams } from './query'

export type TreeNode = { record: RecordData; parent?: string; hasChildren: boolean; matched: boolean; pathUnavailable: boolean }
export type TreePage = { data: TreeNode[]; context?: RecordData[]; meta: RecordListMeta }

export function listTreeRecords(entity: string, relation: 'roots' | 'children' | 'search', params: ListRecordsParams, options: { name?: string; excludeSubtree?: string; signal?: AbortSignal } = {}) {
  const query = buildRecordListQuery(params)
  if (options.name) query.set('name', options.name)
  if (options.excludeSubtree) query.set('exclude-subtree', options.excludeSubtree)
  return apiRequest<TreePage, RecordApiError>(`/api/v1/records/${encodeURIComponent(entity)}/tree/${relation}?${query}`, { signal: options.signal }, {
    error: RecordApiError, fallbackCode: 'tree_failed', invalidResponseMessage: 'Studio could not read the tree.', message: (payload) => payload.error?.message ?? 'Studio could not load the tree.',
  })
}
