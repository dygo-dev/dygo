import test from 'node:test'
import assert from 'node:assert/strict'
import { createPinia, setActivePinia } from 'pinia'

import { useNavigationStore } from './navigation.store.ts'
import { usePreferencesStore } from '../features/preferences/preferences.store.ts'

test('recent pages are isolated by user and logout preserves server preferences', async () => {
  const storage = memoryStorage()
  const hadWindow = 'window' in globalThis
  const previousWindow = globalThis.window
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { localStorage: storage } })
  const originalFetch = globalThis.fetch
  globalThis.fetch = async () => Response.json({ data: {
    'studio.recent-pages': usePreferencesStore().userID === 7
      ? [{ path: '/customers', label: 'Customers', detail: 'Record list' }] : [],
  } })

  try {
    setActivePinia(createPinia())
    const navigation = useNavigationStore()
    const preferences = usePreferencesStore()
    navigation.setRecentUser(7)
    await preferences.startSession(7)
    navigation.rememberRecentPage({ path: '/customers', label: 'Customers', detail: 'Record list' })

    navigation.setRecentUser(8)
    await preferences.startSession(8)
    assert.deepEqual(navigation.recentPages, [])
    navigation.rememberRecentPage({ path: '/orders', label: 'Orders', detail: 'Record list' })

    navigation.setRecentUser(7)
    await preferences.startSession(7)
    assert.deepEqual(navigation.recentPages.map((page) => page.path), ['/customers'])
    navigation.setRecentUser(null)
    assert.deepEqual(navigation.recentPages, [])

    navigation.setRecentUser(7)
    await preferences.startSession(7)
    assert.deepEqual(navigation.recentPages.map((page) => page.path), ['/customers'])
    navigation.setRecentUser(8)
    await preferences.startSession(8)
    assert.deepEqual(navigation.recentPages, [])
  } finally {
    await usePreferencesStore().startSession(null)
    globalThis.fetch = originalFetch
    if (hadWindow) {
      Object.defineProperty(globalThis, 'window', { configurable: true, value: previousWindow })
    } else {
      delete (globalThis as { window?: unknown }).window
    }
  }
})

function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: (key) => values.delete(key),
    clear: () => values.clear(),
    key: (index) => [...values.keys()][index] ?? null,
    get length() { return values.size },
  }
}

test('a page visited during hydration keeps server history', async () => {
  const original = globalThis.fetch
  let resolve!: (response: Response) => void
  globalThis.fetch = () => new Promise<Response>(done => { resolve = done })
  setActivePinia(createPinia())
  const navigation = useNavigationStore()
  try {
    navigation.setRecentUser(7)
    const remember = navigation.rememberRecentPage({ path: '/current', label: 'Current', detail: '' })
    resolve(Response.json({ data: { 'studio.recent-pages': [{ path: '/previous', label: 'Previous', detail: '' }] } }))
    await remember
    assert.deepEqual(navigation.recentPages.map(page => page.path), ['/current', '/previous'])
  } finally {
    navigation.setRecentUser(null)
    globalThis.fetch = original
  }
})
