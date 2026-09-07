import { defineStore } from 'pinia'
import { usePreferencesStore } from '../features/preferences/preferences.store.ts'
import { normalizePinnedItems, pinnedItemID, type PinnedItem } from '../features/pinned/pinned.ts'

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
    pinnedItems: () => normalizePinnedItems(usePreferencesStore().get('studio.pinned-items', [])),
    pinnedOpen: () => usePreferencesStore().get<boolean>('studio.pinned-open', true) !== false,
    pinnedExpanded: () => usePreferencesStore().get<boolean>('studio.pinned-expanded', false) === true,
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

    pin(item: PinnedItem) {
      const id = pinnedItemID(item)
      usePreferencesStore().set('studio.pinned-items', [item, ...this.pinnedItems.filter(candidate => pinnedItemID(candidate) !== id)])
    },

    unpin(item: PinnedItem) {
      const id = pinnedItemID(item)
      usePreferencesStore().set('studio.pinned-items', this.pinnedItems.filter(candidate => pinnedItemID(candidate) !== id))
    },

    togglePin(item: PinnedItem) {
      if (this.pinnedItems.some(candidate => pinnedItemID(candidate) === pinnedItemID(item))) this.unpin(item)
      else this.pin(item)
    },

    reorderPinned(from: number, to: number) {
      if (from === to || from < 0 || to < 0 || from >= this.pinnedItems.length || to >= this.pinnedItems.length) return
      const items = [...this.pinnedItems]
      const [item] = items.splice(from, 1)
      if (!item) return
      items.splice(to, 0, item)
      usePreferencesStore().set('studio.pinned-items', items)
    },

    setPinnedOpen(value: boolean) {
      usePreferencesStore().set('studio.pinned-open', value)
    },

    setPinnedExpanded(value: boolean) {
      usePreferencesStore().set('studio.pinned-expanded', value)
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
