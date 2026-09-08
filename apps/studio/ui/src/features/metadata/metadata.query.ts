import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'

import { getEntityMeta, listEntities } from './metadata.api'
import { useAuthStore } from '@/stores/auth.store'

type QueryToggle = MaybeRefOrGetter<boolean>

export const metadataEntitiesQueryKey = ['metadata', 'entities'] as const

export function metadataEntityMetaQueryKey(entity: string, userID: number | null, sessionVersion: number) {
  return ['metadata', 'entity-meta', entity, userID, sessionVersion] as const
}

export function metadataEntitiesQueryOptions(userID: number | null, sessionVersion: number) {
  return {
    queryKey: [...metadataEntitiesQueryKey, userID, sessionVersion] as const,
    queryFn: ({ signal }: { signal?: AbortSignal }) => listEntities({ signal }),
  }
}

export function metadataEntityMetaQueryOptions(entity: string, userID: number | null, sessionVersion: number) {
  return {
    queryKey: metadataEntityMetaQueryKey(entity, userID, sessionVersion),
    queryFn: ({ signal }: { signal?: AbortSignal }) => getEntityMeta(entity, { signal }),
  }
}

export function useMetadataEntitiesQuery(options: { enabled?: QueryToggle } = {}) {
  const auth = useAuthStore()
  return useQuery(computed(() => ({
    ...metadataEntitiesQueryOptions(auth.currentUser?.id ?? null, auth.sessionVersion),
    enabled: Boolean(auth.currentUser) && toValue(options.enabled ?? true),
  })))
}

export function useMetadataEntityMetaQuery(entity: MaybeRefOrGetter<string>, options: { enabled?: QueryToggle } = {}) {
  const currentEntity = computed(() => toValue(entity).trim())
  const auth = useAuthStore()

  return useQuery(computed(() => ({
    ...metadataEntityMetaQueryOptions(currentEntity.value, auth.currentUser?.id ?? null, auth.sessionVersion),
    enabled: Boolean(auth.currentUser) && currentEntity.value !== '' && toValue(options.enabled ?? true),
  })))
}
