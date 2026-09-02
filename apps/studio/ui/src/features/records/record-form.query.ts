import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery } from '@tanstack/vue-query'

import { queryClient } from '@/app/query'
import { studioSounds } from '@/features/sounds'
import { recordListBaseQueryKey } from './record-list.query'
import {
  createRecord,
  addRecordComment,
  deleteRecord,
  getRecordByName,
  getSingleRecord,
  updateRecord,
  updateSingleRecord,
  type RecordData,
  listRecordActivity,
} from './records.api'

type QueryToggle = MaybeRefOrGetter<boolean>

type CreateRecordVariables = {
  entity: string
  data: RecordData
}

type UpdateRecordVariables = {
  entity: string
  recordName: string
  id: string | number
  data: RecordData
}

type UpdateSingleRecordVariables = {
  entity: string
  data: RecordData
}

type DeleteRecordVariables = {
  entity: string
  recordName: string
  id: string | number
}

type AddRecordCommentVariables = {
  entity: string
  recordID: string | number
  message: string
}

export function recordByNameQueryKey(entity: string, recordName: string) {
  return ['records', 'detail', entity, recordName] as const
}

export function singleRecordQueryKey(entity: string) {
  return ['records', 'single', entity] as const
}

export function recordActivityQueryKey(entity: string, recordID: string | number) {
  return ['records', 'activity', entity, String(recordID)] as const
}

export function useRecordActivityQuery(
  entity: MaybeRefOrGetter<string>,
  recordID: MaybeRefOrGetter<string | number>,
  options: { enabled?: QueryToggle } = {},
) {
  const currentEntity = computed(() => toValue(entity).trim())
  const currentRecordID = computed(() => String(toValue(recordID)).trim())

  return useQuery({
    queryKey: computed(() => recordActivityQueryKey(currentEntity.value, currentRecordID.value)),
    queryFn: ({ signal }) => listRecordActivity(currentEntity.value, currentRecordID.value, { signal }),
    enabled: computed(() => currentEntity.value !== '' && currentRecordID.value !== '0' && toValue(options.enabled ?? true)),
  })
}

export function useAddRecordCommentMutation() {
  return useMutation({
    mutationFn: ({ entity, recordID, message }: AddRecordCommentVariables) => addRecordComment(entity, recordID, message),
    ...studioMutationSoundHandlers<{ created: boolean }, AddRecordCommentVariables>('save', (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: recordActivityQueryKey(variables.entity, variables.recordID) })
    }),
  })
}

export function useRecordByNameQuery(
  entity: MaybeRefOrGetter<string>,
  recordName: MaybeRefOrGetter<string>,
  options: { enabled?: QueryToggle } = {},
) {
  const currentEntity = computed(() => toValue(entity).trim())
  const currentRecordName = computed(() => toValue(recordName).trim())

  return useQuery({
    queryKey: computed(() => recordByNameQueryKey(currentEntity.value, currentRecordName.value)),
    queryFn: ({ signal }) => getRecordByName(currentEntity.value, currentRecordName.value, { signal }),
    enabled: computed(() => (
      currentEntity.value !== ''
      && currentRecordName.value !== ''
      && toValue(options.enabled ?? true)
    )),
  })
}

export function useSingleRecordQuery(
  entity: MaybeRefOrGetter<string>,
  options: { enabled?: QueryToggle } = {},
) {
  const currentEntity = computed(() => toValue(entity).trim())

  return useQuery({
    queryKey: computed(() => singleRecordQueryKey(currentEntity.value)),
    queryFn: ({ signal }) => getSingleRecord(currentEntity.value, { signal }),
    enabled: computed(() => currentEntity.value !== '' && toValue(options.enabled ?? true)),
  })
}

export function useCreateRecordMutation() {
  return useMutation({
    mutationFn: ({ entity, data }: CreateRecordVariables) => createRecord(entity, data),
    ...studioMutationSoundHandlers<RecordData, CreateRecordVariables>('save', (record, variables) => {
      cacheNamedRecord(variables.entity, record)
      invalidateRecordLists(variables.entity)
    }),
  })
}

export function useUpdateRecordMutation() {
  return useMutation({
    mutationFn: ({ entity, id, data }: UpdateRecordVariables) => updateRecord(entity, id, data),
    ...studioMutationSoundHandlers<RecordData, UpdateRecordVariables>('save', (record, variables) => {
      queryClient.setQueryData(recordByNameQueryKey(variables.entity, variables.recordName), record)
      cacheNamedRecord(variables.entity, record)
      invalidateRecordLists(variables.entity)
    }),
  })
}

export function useUpdateSingleRecordMutation() {
  return useMutation({
    mutationFn: ({ entity, data }: UpdateSingleRecordVariables) => updateSingleRecord(entity, data),
    ...studioMutationSoundHandlers<RecordData, UpdateSingleRecordVariables>('save', (record, variables) => {
      queryClient.setQueryData(singleRecordQueryKey(variables.entity), record)
      cacheNamedRecord(variables.entity, record)
      invalidateRecordLists(variables.entity)
    }),
  })
}

export function useDeleteRecordMutation() {
  return useMutation({
    mutationFn: ({ entity, id }: DeleteRecordVariables) => deleteRecord(entity, id),
    ...studioMutationSoundHandlers<void, DeleteRecordVariables>('delete', (_result, variables) => {
      queryClient.removeQueries({
        queryKey: recordByNameQueryKey(variables.entity, variables.recordName),
        exact: true,
      })
      invalidateRecordLists(variables.entity)
    }),
  })
}

type StudioMutationSound = 'save' | 'delete'

function studioMutationSoundHandlers<TData, TVariables>(
  sound: StudioMutationSound,
  onSuccess?: (data: TData, variables: TVariables) => void,
) {
  return {
    onSuccess: (data: TData, variables: TVariables) => {
      if (sound === 'delete') {
        studioSounds.delete()
      } else {
        studioSounds.save()
      }

      onSuccess?.(data, variables)
    },
    onError: () => {
      studioSounds.error()
    },
  }
}

function cacheNamedRecord(entity: string, record: RecordData) {
  if (typeof record.name !== 'string' || record.name.length === 0) {
    return
  }

  queryClient.setQueryData(recordByNameQueryKey(entity, record.name), record)
}

function invalidateRecordLists(entity: string) {
  void queryClient.invalidateQueries({ queryKey: recordListBaseQueryKey(entity) })
}
