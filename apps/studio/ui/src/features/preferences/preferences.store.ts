import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiRequest, ApiClientError, type DataEnvelope } from '../api/client.ts'
import { useToastStore } from '../toasts/toasts.store.ts'
import { bindStudioThemePreference, isStudioThemePreference, setStudioThemePreference } from '../theme/studio-theme.ts'
import { bindStudioSoundPreference, setStudioSoundEnabled } from '../sounds/studio-sounds.ts'

class PreferenceError extends ApiClientError {
  constructor(code: string, message: string) { super('PreferenceError', code, message) }
}

export const usePreferencesStore = defineStore('preferences', () => {
  const values = ref<Record<string, unknown>>({})
  const ready = ref(false)
  const error = ref<string | null>(null)
  const user = ref<number | null>(null)
  let generation = 0
  let controller = new AbortController()
  let hydration: Promise<void> = Promise.resolve()
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  const queues = new Map<string, Promise<void>>()
  const pending = new Map<string, unknown>()
  const scheduled = new Map<string, unknown>()

  function request<T>(key = '', init: RequestInit = {}) {
    return apiRequest<DataEnvelope<T>, PreferenceError>(
      `/api/v1/studio/preferences${key ? `/${encodeURIComponent(key)}` : ''}`,
      { ...init, signal: controller.signal },
      { error: PreferenceError, fallbackCode: 'preferences_failed', invalidResponseMessage: 'Studio could not read preferences.', message: p => p.error?.message ?? 'Studio could not save preferences.' },
    )
  }

  function report(cause: unknown) {
    error.value = cause instanceof Error ? cause.message : 'Studio could not save preferences.'
    useToastStore().show({ title: 'Preferences could not sync', content: error.value, type: 'danger' })
  }

  function get<T>(key: string, fallback: T): T {
    return Object.hasOwn(values.value, key) ? values.value[key] as T : fallback
  }

  function applyAppearance() {
    const theme = values.value['studio.theme']
    setStudioThemePreference(isStudioThemePreference(theme) ? theme : 'system', undefined, false)
    const sound = values.value['studio.sounds']
    setStudioSoundEnabled(typeof sound === 'boolean' ? sound : true, false)
  }

  function schedule(key: string, value: unknown) {
    clearTimeout(timers.get(key))
    scheduled.set(key, value)
    timers.set(key, setTimeout(() => enqueue(key), 250))
  }

  function enqueue(key: string) {
    clearTimeout(timers.get(key))
    timers.delete(key)
    const value = scheduled.get(key)
    scheduled.delete(key)
    const session = generation
    const write = (queues.get(key) ?? Promise.resolve()).then(async () => {
      if (session !== generation || !ready.value) return
      try {
        await request(key, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ value }) })
      } catch (cause) {
        if (session === generation) report(cause)
      }
    })
    queues.set(key, write)
    void write.finally(() => { if (queues.get(key) === write) queues.delete(key) })
  }

  async function flush() {
    const session = generation
    await hydration
    if (session !== generation) return
    for (const key of scheduled.keys()) enqueue(key)
    await Promise.all(queues.values())
  }

  function set(key: string, value: unknown) {
    if (user.value === null) return
    values.value = { ...values.value, [key]: value }
    if (key === 'studio.theme' || key === 'studio.sounds') applyAppearance()
    if (ready.value) schedule(key, value)
    else {
      pending.set(key, value)
      if (error.value) void hydrate()
    }
  }

  async function importMissing(legacy: Record<string, unknown>) {
    const session = generation
    await hydration
    if (session !== generation || !ready.value) return
    for (const [key, value] of Object.entries(legacy)) {
      if (value !== undefined && !Object.hasOwn(values.value, key)) set(key, value)
    }
    applyAppearance()
  }

  function startSession(userID: number | null): Promise<void> {
    if (user.value === userID) return hydration
    generation++
    controller.abort()
    controller = new AbortController()
    for (const timer of timers.values()) clearTimeout(timer)
    timers.clear()
    scheduled.clear()
    queues.clear()
    pending.clear()
    user.value = userID
    values.value = {}
    ready.value = false
    error.value = null
    bindStudioThemePreference(userID === null ? null : value => set('studio.theme', value))
    bindStudioSoundPreference(userID === null ? null : value => set('studio.sounds', value))
    if (userID === null) {
      setStudioSoundEnabled(true, false)
      return hydration = Promise.resolve()
    }
    return hydrate()
  }

  function hydrate(): Promise<void> {
    const session = generation
    error.value = null
    hydration = request<Record<string, unknown>>().then(({ data }) => {
      if (session !== generation) return
      values.value = { ...data, ...Object.fromEntries(pending) }
      ready.value = true
      for (const [key, value] of pending) schedule(key, value)
      pending.clear()
      applyAppearance()
    }).catch(cause => { if (session === generation) report(cause) })
    return hydration
  }

  return { ready, error, userID: computed(() => user.value), get, set, importMissing, startSession, flush }
})
