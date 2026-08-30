import { createRouter, createWebHistory, type RouteLocationRaw, type Router } from 'vue-router'

import LoginPage from '@/features/auth/LoginPage.vue'
import type { StudioBoot, StudioPageClaim } from '@/features/boot/boot.api'
import PageHost from '@/features/pages/PageHost.vue'
import NotFoundPage from '@/pages/NotFoundPage.vue'
import RecordFormPage from '@/pages/RecordFormPage.vue'
import RecordsPage from '@/pages/RecordsPage.vue'
import { useAuthStore } from '@/stores/auth.store'
import { useBootStore } from '@/stores/boot.store'
import { pinia } from '@/stores/pinia'
import { normalizePageClaimPath, pageRouteName, routeParam, RouteName } from './routes'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    redirectIfAuthenticated?: boolean
    pageClaim?: StudioPageClaim
  }
}

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: RouteName.Login,
      component: LoginPage,
      meta: { public: true, redirectIfAuthenticated: true },
    },
    {
      path: '/',
      name: RouteName.Home,
      component: PageHost,
      props: { app: 'studio', pageKey: 'home' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: RouteName.NotFound,
      component: NotFoundPage,
    },
  ],
})

let entityRoutesInstalled = false
let pageRouteFingerprint = ''
let removePageRoutes: Array<() => void> = []

router.beforeEach(async (to): Promise<RouteLocationRaw | undefined> => {
  const authStore = useAuthStore(pinia)
  const bootStore = useBootStore(pinia)
  const user = await authStore.loadCurrentUser()

  if (!user) {
    bootStore.clearBoot()
    if (to.meta.public) {
      return undefined
    }
    return {
      name: RouteName.Login,
      query: { redirect: to.fullPath },
    }
  }

  const boot = await bootStore.loadBoot()

  if (!boot && bootStore.status === 'unauthenticated') {
    authStore.clearSession()
    return {
      name: RouteName.Login,
      query: { redirect: to.fullPath },
    }
  }
  const routesChanged = boot ? installBootRoutes(router, boot.pages) : false

  if (to.meta.redirectIfAuthenticated) {
    return bootHomeRedirect(boot) ?? { name: RouteName.Home }
  }

  if (!boot) {
    return undefined
  }

  if (routesChanged && to.name === RouteName.NotFound) {
    return to.fullPath
  }

  if (to.name === RouteName.Home) {
    return bootHomeRedirect(boot)
  }

  return undefined
})

export function installBootRoutes(target: Router, claims: StudioPageClaim[] = []): boolean {
  const normalized = claims
    .map((claim) => ({ ...claim, path: normalizePageClaimPath(claim.path) }))
    .filter((claim) => claim.app.trim() !== '' && claim.key.trim() !== '' && claim.path !== null)
    .map((claim) => ({ ...claim, path: claim.path as string }))
    .filter((claim) => claim.path !== '/' && claim.path !== '/login')
    .sort((left, right) => left.path.localeCompare(right.path))

  const fingerprint = JSON.stringify(normalized)
  let changed = false
  if (fingerprint !== pageRouteFingerprint) {
    for (const removeRoute of removePageRoutes) {
      removeRoute()
    }
    removePageRoutes = normalized.map((claim) => target.addRoute({
      path: claim.path,
      name: pageRouteName(claim.app, claim.key),
      component: PageHost,
      props: { app: claim.app, pageKey: claim.key },
      meta: { pageClaim: claim },
    }))
    pageRouteFingerprint = fingerprint
    changed = true
  }

  if (!entityRoutesInstalled) {
    installEntityRoutes(target)
    entityRoutesInstalled = true
    changed = true
  }

  return changed
}

function installEntityRoutes(target: Router) {
  target.addRoute({
    path: '/:entity/new',
    name: RouteName.RecordNew,
    component: RecordFormPage,
    props: (route) => ({
      entity: routeParam(route.params.entity as string | string[]),
      mode: 'new',
    }),
  })
  target.addRoute({
    path: '/:entity/:recordName',
    name: RouteName.RecordDetail,
    component: RecordFormPage,
    props: (route) => ({
      entity: routeParam(route.params.entity as string | string[]),
      recordName: routeParam(route.params.recordName as string | string[]),
      mode: 'record',
    }),
  })
  target.addRoute({
    path: '/:entity',
    name: RouteName.EntityRecords,
    component: RecordsPage,
    props: (route) => ({ entity: routeParam(route.params.entity as string | string[]) }),
  })
}

function bootHomeRedirect(boot: StudioBoot | null): RouteLocationRaw | undefined {
  const home = normalizeBootHome(boot?.defaults.home)
  return home === '/' ? undefined : home
}

function normalizeBootHome(value: unknown): string {
  return normalizePageClaimPath(value) ?? '/'
}
