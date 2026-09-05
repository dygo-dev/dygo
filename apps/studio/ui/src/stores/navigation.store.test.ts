import test from 'node:test'
import assert from 'node:assert/strict'
import { createPinia, setActivePinia } from 'pinia'

import { useNavigationStore } from './navigation.store.ts'

test('recent pages are isolated by user and cleared on logout', () => {
  const storage = memoryStorage()
  const hadWindow = 'window' in globalThis
  const previousWindow = globalThis.window
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { localStorage: storage } })

  try {
    setActivePinia(createPinia())
    const navigation = useNavigationStore()
    navigation.setRecentUser(7)
    navigation.rememberRecentPage({ path: '/customers', label: 'Customers', detail: 'Record list' })

    navigation.setRecentUser(8)
    assert.deepEqual(navigation.recentPages, [])
    navigation.rememberRecentPage({ path: '/orders', label: 'Orders', detail: 'Record list' })

    navigation.setRecentUser(7)
    assert.deepEqual(navigation.recentPages.map((page) => page.path), ['/customers'])
    navigation.setRecentUser(null)
    assert.deepEqual(navigation.recentPages, [])

    navigation.setRecentUser(7)
    assert.deepEqual(navigation.recentPages, [])
    navigation.setRecentUser(8)
    assert.deepEqual(navigation.recentPages.map((page) => page.path), ['/orders'])
  } finally {
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
