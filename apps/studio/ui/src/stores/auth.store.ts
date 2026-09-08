import { defineStore } from 'pinia'

import { AuthApiError, getCurrentUser, logout as logoutRequest, type CurrentUser } from '@/features/auth/auth.api'
import { statusForError, storeError, type LoadStatus, type StoreError } from './status'
import { useNavigationStore } from './navigation.store'
import { usePreferencesStore } from '../features/preferences/preferences.store'

type LoadCurrentUserOptions = {
  force?: boolean
}

type AuthState = {
  currentUser: CurrentUser | null
  status: LoadStatus
  error: StoreError | null
  loaded: boolean
  pendingUser: Promise<CurrentUser | null> | null
  sessionVersion: number
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    currentUser: null,
    status: 'idle',
    error: null,
    loaded: false,
    pendingUser: null,
    sessionVersion: 0,
  }),

  actions: {
    setCurrentUser(user: CurrentUser | null) {
      this.sessionVersion++
      this.currentUser = user
      this.loaded = true
      this.error = null
      this.status = user ? 'ready' : 'unauthenticated'
      this.pendingUser = null
      useNavigationStore().setRecentUser(user?.id ?? null)
      if (user) {
        const legacy: Record<string, unknown> = {}
        try {
          const storage = window.localStorage
          const owner = storage.getItem('studio:legacy-preference-owner')
          if (owner === null) storage.setItem('studio:legacy-preference-owner', String(user.id))
          if ((owner === null || owner === String(user.id)) && storage.getItem('studio:legacy-preferences-imported') !== 'true') {
            const theme = storage.getItem('studio:theme')
            const sounds = storage.getItem('studio:sounds-enabled')
            if (theme !== null) legacy['studio.theme'] = theme
            if (sounds !== null) legacy['studio.sounds'] = sounds !== 'false'
          }
        } catch {
          // Server preferences also work when browser storage is unavailable.
        }
        const preferences = usePreferencesStore()
        void preferences.importMissing(legacy).then(async () => {
          await preferences.flush()
          if (preferences.userID !== user.id || !preferences.ready || preferences.error) return
          try {
            window.localStorage.setItem('studio:legacy-preferences-imported', 'true')
          } catch {
            // Browser import markers are optional.
          }
        })
      }
    },

    clearSession() {
      this.setCurrentUser(null)
    },

    async logout(): Promise<void> {
      try {
        await usePreferencesStore().flush()
        await logoutRequest()
      } finally {
        this.clearSession()
      }
    },

    async loadCurrentUser(options: LoadCurrentUserOptions = {}): Promise<CurrentUser | null> {
      if (this.loaded && !options.force) {
        return this.currentUser
      }

      if (this.pendingUser && !options.force) {
        return this.pendingUser
      }

      this.status = 'loading'
      this.error = null
      const version = ++this.sessionVersion

      this.pendingUser = getCurrentUser()
        .then((user) => {
          if (version !== this.sessionVersion) return this.currentUser
          this.setCurrentUser(user)
          return user
        })
        .catch((error: unknown) => {
          if (version !== this.sessionVersion) return this.currentUser
          const normalized = storeError(error, 'Studio could not read the current session.')
          useNavigationStore().setRecentUser(null)
          this.currentUser = null
          this.loaded = true
          this.error = normalized
          this.status = error instanceof AuthApiError ? statusForError(normalized) : 'error'
          return null
        })
        .finally(() => {
          if (version === this.sessionVersion) this.pendingUser = null
        })

      return this.pendingUser
    },
  },
})
