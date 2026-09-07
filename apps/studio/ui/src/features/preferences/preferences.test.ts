import assert from 'node:assert/strict'
import test from 'node:test'
import { setTimeout as delay } from 'node:timers/promises'
import { createPinia, setActivePinia } from 'pinia'
import { usePreferencesStore } from './preferences.store.ts'
import { useToastStore } from '../toasts/toasts.store.ts'
import { getStudioThemePreference } from '../theme/studio-theme.ts'
import { isStudioSoundEnabled } from '../sounds/studio-sounds.ts'

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(done => { resolve = done })
  return { promise, resolve }
}

test('an account without appearance preferences gets defaults, not the prior user settings', async () => {
  const originalFetch = globalThis.fetch
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const storage = new Map<string, string>()
  Object.defineProperty(globalThis, 'window', { configurable: true, value: {
    localStorage: { getItem: (key: string) => storage.get(key) ?? null, setItem: (key: string, value: string) => storage.set(key, value) },
    matchMedia: () => ({ matches: false }),
  } })
  let calls = 0
  globalThis.fetch = async () => Response.json({ data: ++calls === 1 ? { 'studio.theme': 'dark', 'studio.sounds': false } : {} })
  setActivePinia(createPinia())
  const store = usePreferencesStore()
  try {
    await store.startSession(7)
    assert.equal(getStudioThemePreference(), 'dark')
    assert.equal(isStudioSoundEnabled(), false)
    await store.startSession(null)
    assert.equal(getStudioThemePreference(), 'dark')
    await store.startSession(8)
    assert.equal(getStudioThemePreference(), 'system')
    assert.equal(isStudioSoundEnabled(), true)
  } finally {
    await store.startSession(null)
    globalThis.fetch = originalFetch
    if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow)
    else delete (globalThis as { window?: unknown }).window
  }
})

test('hydration preserves server values and imports only missing keys', async () => {
  const original = globalThis.fetch
  const response = deferred<Response>()
  const writes: unknown[] = []
  globalThis.fetch = async (_url, init) => {
    if (!init?.method) return response.promise
    writes.push(JSON.parse(String(init.body)))
    return Response.json({ data: {} })
  }
  setActivePinia(createPinia())
  const store = usePreferencesStore()
  try {
    const loading = store.startSession(7)
    const importing = store.importMissing({ 'studio.theme': 'light', 'studio.records.page-size': 50 })
    assert.equal(store.ready, false)
    assert.equal(writes.length, 0)
    response.resolve(Response.json({ data: { 'studio.theme': 'dark' } }))
    await loading
    await importing
    await delay(300)
    assert.equal(store.get('studio.theme', ''), 'dark')
    assert.deepEqual(writes, [{ value: 50 }])
  } finally { await store.startSession(null); globalThis.fetch = original }
})

test('writes debounce and serialize per key; logout cancels queued work', async () => {
  const original = globalThis.fetch
  const first = deferred<Response>()
  const writes: number[] = []
  globalThis.fetch = async (_url, init) => {
    if (!init?.method) return Response.json({ data: {} })
    writes.push(JSON.parse(String(init.body)).value)
    return writes.length === 1 ? first.promise : Response.json({ data: {} })
  }
  setActivePinia(createPinia())
  const store = usePreferencesStore()
  try {
    await store.startSession(7)
    store.set('studio.records.page-size', 20)
    store.set('studio.records.page-size', 50)
    await delay(300)
    assert.deepEqual(writes, [50])
    store.set('studio.records.page-size', 100)
    await delay(300)
    assert.deepEqual(writes, [50])
    await store.startSession(null)
    first.resolve(Response.json({ data: {} }))
    await delay(0)
    assert.deepEqual(writes, [50])
    assert.equal(store.get('studio.records.page-size', 20), 20)
  } finally { first.resolve(Response.json({ data: {} })); await store.startSession(null); globalThis.fetch = original }
})

test('late hydration, legacy import, and flush cannot operate on another session', async () => {
  const original = globalThis.fetch
  const first = deferred<Response>()
  let calls = 0
  const writes: unknown[] = []
  globalThis.fetch = async (_url, init) => {
    if (init?.method) {
      writes.push(JSON.parse(String(init.body)))
      return Response.json({ data: {} })
    }
    return ++calls === 1 ? first.promise : Response.json({ data: { marker: 'second' } })
  }
  setActivePinia(createPinia())
  const store = usePreferencesStore()
  try {
    const stale = store.startSession(7)
    const staleImport = store.importMissing({ marker: 'legacy' })
    const staleFlush = store.flush()
    await store.startSession(8)
    store.set('studio.records.page-size', 50)
    first.resolve(Response.json({ data: { marker: 'first' } }))
    await stale
    await staleImport
    await staleFlush
    assert.equal(store.get('marker', ''), 'second')
    assert.deepEqual(writes, [])
    await store.flush()
    assert.deepEqual(writes, [{ value: 50 }])
  } finally { await store.startSession(null); globalThis.fetch = original }
})

test('failed hydration recovers on the next edit and preserves pending edits', async () => {
  const original = globalThis.fetch
  let reads = 0
  const writes: unknown[] = []
  globalThis.fetch = async (_url, init) => {
    if (init?.method) {
      writes.push(JSON.parse(String(init.body)))
      return Response.json({ data: {} })
    }
    if (++reads === 1) throw new Error('Network unavailable')
    return Response.json({ data: { 'studio.theme': 'dark', 'studio.records.page-size': 20 } })
  }
  setActivePinia(createPinia())
  const store = usePreferencesStore()
  const toasts = useToastStore()
  try {
    const loading = store.startSession(7)
    store.set('studio.records.page-size', 50)
    await loading
    await store.importMissing({ 'studio.theme': 'light' })
    assert.equal(store.ready, false)
    assert.equal(store.get('studio.theme', 'system'), 'system')
    assert.equal(toasts.toasts[0]?.type, 'danger')
    assert.equal(reads, 1)
    store.set('studio.sidebar-collapsed', true)
    await store.flush()
    assert.equal(reads, 2)
    assert.equal(store.ready, true)
    assert.equal(store.error, null)
    assert.equal(store.get('studio.theme', ''), 'dark')
    assert.equal(store.get('studio.records.page-size', 20), 50)
    assert.deepEqual(writes, [{ value: 50 }, { value: true }])
  } finally {
    for (const toast of toasts.toasts) toasts.dismiss(toast.id)
    await store.startSession(null)
    globalThis.fetch = original
  }
})
