import test from 'node:test'
import assert from 'node:assert/strict'

import { getNotificationDeepLink, getUnreadNotifications, markNotificationRead } from './notifications.api.ts'

test('notification API uses authenticated inbox endpoints', async (t) => {
  const originalFetch = globalThis.fetch
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  const calls: string[] = []
  globalThis.fetch = (async (input, init) => {
    calls.push(`${init?.method} ${String(input)}`)
    if (String(input).endsWith('/read')) {
      return new Response(JSON.stringify({ data: notification }))
    }
    if (String(input).endsWith('/deep-link')) {
      return new Response(JSON.stringify({ data: { 'deep-link': '/hr-leave-request/HRL-1' } }))
    }
    return new Response(JSON.stringify({ data: [notification], meta: { limit: 20, count: 1 } }))
  }) as typeof fetch

  const items = await getUnreadNotifications()
  await markNotificationRead(4)
  const link = await getNotificationDeepLink(4)

  assert.equal(items[0]?.title, 'Leave approved')
  assert.equal(link, '/hr-leave-request/HRL-1')
  assert.deepEqual(calls, [
    'GET /api/v1/notifications?limit=20',
    'POST /api/v1/notifications/4/read',
    'GET /api/v1/notifications/4/deep-link',
  ])
})

const notification = {
  id: 4,
  name: 'notice-4',
  title: 'Leave approved',
  message: 'Your leave was approved.',
  'deep-link': '/hr-leave-request/HRL-1',
  'created-at': '2026-08-31T12:00:00Z',
  'read-at': null,
}

