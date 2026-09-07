import { defineStore } from 'pinia'
import { usePreferencesStore } from '../features/preferences/preferences.store.ts'

export type RecentPage = {
  path: string
  label: string
  detail: string
}

const RECENT_PAGES_STORAGE_KEY = 'dygo.studio.recentPages'
const MAX_RECENT_PAGES = 10

export const useNavigationStore = defineStore('navigation', {
  state: () => ({
    recentUserID: null as number | null,
    recentGeneration: 0,
    commandMenuOpen: false,
    routeReloadVersion: 0,
  }),

  getters: {
    sidebarCollapsed: () => usePreferencesStore().get<boolean>('studio.sidebar-collapsed', false) === true,
    recentPages: () => normalizeRecentPages(usePreferencesStore().get('studio.recent-pages', [])),
  },

  actions: {
    setRecentUser(userID: number | null) {
      this.recentGeneration++
      this.recentUserID = userID
      this.commandMenuOpen = false
      const preferences = usePreferencesStore()
      void preferences.startSession(userID)
      if (userID !== null) void preferences.importMissing({ 'studio.recent-pages': readRecentPages(userID) })
    },

    clearRecentPages() {
      usePreferencesStore().set('studio.recent-pages', [])
    },

    setSidebarCollapsed(value: boolean) {
      usePreferencesStore().set('studio.sidebar-collapsed', value)
    },

    toggleSidebar() {
      this.setSidebarCollapsed(!this.sidebarCollapsed)
    },

    openCommandMenu() {
      this.commandMenuOpen = true
    },

    closeCommandMenu() {
      this.commandMenuOpen = false
    },

    requestRouteReload() {
      this.routeReloadVersion += 1
    },

    async rememberRecentPage(page: RecentPage | null) {
      if (!page || page.path.trim() === '' || page.label.trim() === '') {
        return
      }

      const generation = this.recentGeneration
      const preferences = usePreferencesStore()
      await preferences.startSession(this.recentUserID)
      if (generation !== this.recentGeneration || !preferences.ready) return

      const pages = [
        page,
        ...this.recentPages.filter((recentPage) => recentPage.path !== page.path),
      ].slice(0, MAX_RECENT_PAGES)

      usePreferencesStore().set('studio.recent-pages', pages)
    },
  },
})

function recentPagesKey(userID: number): string {
  return `${RECENT_PAGES_STORAGE_KEY}.${userID}`
}

function readRecentPages(userID: number): RecentPage[] {
  try {
    return normalizeRecentPages(JSON.parse(window.localStorage.getItem(recentPagesKey(userID)) ?? '[]'))
  } catch {
    return []
  }
}

function normalizeRecentPages(value: unknown): RecentPage[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item): RecentPage | null => {
      if (!item || typeof item !== 'object') {
        return null
      }

      const path = typeof item.path === 'string' ? item.path : ''
      const label = typeof item.label === 'string' ? item.label : ''
      const detail = typeof item.detail === 'string' ? item.detail : ''
      if (!path || !label) {
        return null
      }

      return { path, label, detail }
    })
    .filter((item): item is RecentPage => Boolean(item))
    .slice(0, MAX_RECENT_PAGES)
}
