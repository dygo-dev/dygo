import test from 'node:test'
import assert from 'node:assert/strict'

import { getPage, PageApiError } from './pages.api.ts'

test('getPage requests the exact app and Page key', async (t) => {
  const originalFetch = globalThis.fetch
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  globalThis.fetch = (async (input, init) => {
    assert.equal(input, '/api/v1/pages/studio/home')
    assert.equal(init?.method, 'GET')
    assert.equal(init?.credentials, 'include')
    return new Response(JSON.stringify({
      data: {
        name: 'studio.home',
        key: 'home',
        label: 'Home',
        description: 'Start page',
        icon: 'house',
        path: '/',
        renderer: 'entity-index',
        options: {},
        app: { name: 'studio', label: 'Studio' },
      },
    }))
  }) as typeof fetch

  const page = await getPage('studio', 'home')
  assert.equal(page.renderer, 'entity-index')
  assert.equal(page.path, '/')
})

test('getPage keeps forbidden as a Page API error', async (t) => {
  const originalFetch = globalThis.fetch
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  globalThis.fetch = (async () => new Response(JSON.stringify({
    error: { code: 'forbidden', message: 'forbidden' },
  }), { status: 403 })) as typeof fetch

  await assert.rejects(
    getPage('studio', 'home'),
    (error: unknown) => error instanceof PageApiError
      && error.code === 'forbidden'
      && error.message === 'You do not have access to this Page.',
  )
})
