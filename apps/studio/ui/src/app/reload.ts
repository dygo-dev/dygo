import type { Router } from 'vue-router'

import { queryClient } from '@/app/query'
import { metadataEntitiesQueryOptions } from '@/features/metadata/metadata.query'
import { platformConfigQueryOptions } from '@/features/platform/platform.query'
import { RouteName } from '@/router/routes'
import { useAuthStore } from '@/stores/auth.store'
import { useBootStore } from '@/stores/boot.store'
import { useNavigationStore } from '@/stores/navigation.store'
import { usePreferencesStore } from '@/features/preferences/preferences.store'
import { pinia } from '@/stores/pinia'

export async function reloadStudioApp(router: Router): Promise<void> {
  const currentRoute = router.currentRoute.value
  const redirectTarget = currentRoute.fullPath
  const authStore = useAuthStore(pinia)
  const bootStore = useBootStore(pinia)
  const navigationStore = useNavigationStore(pinia)

  await usePreferencesStore(pinia).flush()
  authStore.clearSession()
  bootStore.$reset()
  navigationStore.$reset()
  queryClient.clear()

  const user = await authStore.loadCurrentUser({ force: true })
  if (!user) {
    await router.replace({ name: RouteName.Login, query: { redirect: redirectTarget } })
    return
  }

  const boot = await bootStore.loadBoot({ force: true })
  await Promise.allSettled([
    queryClient.fetchQuery(platformConfigQueryOptions()),
    queryClient.fetchQuery(metadataEntitiesQueryOptions(authStore.currentUser?.id ?? null, authStore.sessionVersion)),
  ])

  navigationStore.requestRouteReload()

  if (currentRoute.name === RouteName.Home) {
    const home = normalizeReloadHome(boot?.defaults.home)
    if (home !== '/') {
      await router.replace(home)
    }
  }
}

function normalizeReloadHome(value: unknown): string {
  if (typeof value !== 'string') {
    return '/'
  }

  const home = value.trim()
  if (home === '' || !home.startsWith('/') || home.startsWith('//')) {
    return '/'
  }

  return home
}
