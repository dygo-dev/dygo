import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'

import { getPage } from './pages.api'

export function pageQueryKey(app: string, key: string) {
  return ['pages', app, key] as const
}

export function usePageQuery(app: MaybeRefOrGetter<string>, key: MaybeRefOrGetter<string>) {
  const currentApp = computed(() => toValue(app).trim())
  const currentKey = computed(() => toValue(key).trim())

  return useQuery({
    queryKey: computed(() => pageQueryKey(currentApp.value, currentKey.value)),
    queryFn: ({ signal }) => getPage(currentApp.value, currentKey.value, { signal }),
    enabled: computed(() => currentApp.value !== '' && currentKey.value !== ''),
  })
}
