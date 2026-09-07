import test from 'node:test'
import assert from 'node:assert/strict'

import {
  ApiClientError,
  apiRequest,
  setAPIDialogHandler,
  setAPIToastHandler,
  type ApiErrorEnvelope,
  type DataEnvelope,
} from './client.ts'

class TestApiError extends ApiClientError {
  constructor(code: string, message: string, details?: Record<string, unknown>) {
    super('TestApiError', code, message, details)
  }
}

test('apiRequest applies credentials and returns successful envelopes', async (t) => {
  let observedCredentials: RequestCredentials | undefined
  t.mock.method(globalThis, 'fetch', async (_input: RequestInfo | URL, init?: RequestInit) => {
    observedCredentials = init?.credentials
    return Response.json({ data: { ok: true } })
  })

  const payload = await apiRequest<DataEnvelope<{ ok: boolean }>, TestApiError>('/api/test', { method: 'GET' }, requestOptions())

  assert.deepEqual(payload.data, { ok: true })
  assert.equal(observedCredentials, 'include')
})

for (const [kind, setHandler] of [
  ['dialog', setAPIDialogHandler],
  ['toast', setAPIToastHandler],
] as const) {
  for (const handlerFails of [false, true]) {
    const handlerState = handlerFails ? 'throwing' : 'working'

    test(`apiRequest returns success with a ${handlerState} ${kind} handler`, async (t) => {
      t.after(() => setHandler(null))
      const titles: string[] = []
      setHandler((message) => {
        titles.push(message.title)
        if (handlerFails) throw new Error(`${kind} failed`)
      })
      t.mock.method(globalThis, 'fetch', async () => Response.json({
        data: { ok: true },
        [kind]: { title: 'Saved' },
      }))

      const payload = await apiRequest<DataEnvelope<{ ok: boolean }>, TestApiError>('/api/test', { method: 'GET' }, requestOptions())

      assert.deepEqual(payload.data, { ok: true })
      assert.deepEqual(titles, ['Saved'])
    })

    test(`apiRequest emits ${kind} before rejection with a ${handlerState} handler`, async (t) => {
      t.after(() => setHandler(null))
      const titles: string[] = []
      setHandler((message) => {
        titles.push(message.title)
        if (handlerFails) throw new Error(`${kind} failed`)
      })
      t.mock.method(globalThis, 'fetch', async () => Response.json({
        error: {
          code: 'forbidden',
          message: 'permission denied',
          [kind]: { title: 'Access denied' },
        },
      }, { status: 403 }))

      await assert.rejects(
        apiRequest<DataEnvelope<unknown>, TestApiError>('/api/test', { method: 'GET' }, requestOptions()),
        (error) => {
          assert.deepEqual(titles, ['Access denied'])
          assert.ok(error instanceof TestApiError)
          assert.equal(error.code, 'forbidden')
          assert.equal(error.message, 'mapped: permission denied')
          return true
        },
      )
    })
  }
}

test('apiRequest maps error envelopes through the domain error class', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({
    error: { code: 'forbidden', message: 'permission denied', details: { entity: 'user' } },
  }, { status: 403 }))

  await assert.rejects(
    apiRequest<DataEnvelope<unknown>, TestApiError>('/api/test', { method: 'GET' }, requestOptions()),
    (error) => {
      assert.ok(error instanceof TestApiError)
      assert.equal(error.code, 'forbidden')
      assert.deepEqual(error.details, { entity: 'user' })
      assert.equal(error.message, 'mapped: permission denied')
      return true
    },
  )
})

test('apiRequest reports invalid JSON with the domain error class', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => new Response('not json', { status: 200 }))

  await assert.rejects(
    apiRequest<DataEnvelope<unknown>, TestApiError>('/api/test', { method: 'GET' }, requestOptions()),
    (error) => {
      assert.ok(error instanceof TestApiError)
      assert.equal(error.code, 'invalid_response')
      assert.equal(error.message, 'invalid response')
      return true
    },
  )
})

function requestOptions() {
  return {
    error: TestApiError,
    fallbackCode: 'request_failed',
    invalidResponseMessage: 'invalid response',
    message: (payload: ApiErrorEnvelope) => `mapped: ${payload.error?.message ?? 'failed'}`,
  }
}
