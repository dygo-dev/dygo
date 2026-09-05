import { defineStore } from 'pinia'

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
    sidebarCollapsed: false,
    commandMenuOpen: false,
    routeReloadVersion: 0,
    recentPages: [] as RecentPage[],
  }),

  actions: {
    setRecentUser(userID: number | null) {
      if (this.recentUserID === userID) {
        if (userID === null) {
          this.clearRecentPages()
        }
        return
      }

      if (userID === null) {
        this.clearRecentPages()
      }
      this.recentUserID = userID
      this.recentPages = userID === null ? [] : readRecentPages(userID)
    },

    clearRecentPages() {
      if (this.recentUserID !== null) {
        try {
          window.localStorage.removeItem(recentPagesKey(this.recentUserID))
        } catch {
          // Local preferences are best-effort.
        }
      }
      this.recentPages = []
    },

    setSidebarCollapsed(value: boolean) {
      this.sidebarCollapsed = value
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
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

    rememberRecentPage(page: RecentPage | null) {
      if (!page || page.path.trim() === '' || page.label.trim() === '') {
        return
      }

      this.recentPages = [
        page,
        ...this.recentPages.filter((recentPage) => recentPage.path !== page.path),
      ].slice(0, MAX_RECENT_PAGES)

      if (this.recentUserID === null) {
        return
      }

      try {
        window.localStorage.setItem(recentPagesKey(this.recentUserID), JSON.stringify(this.recentPages))
      } catch {
        // Local preferences are best-effort.
      }
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
