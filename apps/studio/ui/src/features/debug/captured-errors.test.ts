import assert from 'node:assert/strict'
import test from 'node:test'
import { createRenderer, defineComponent, h, onErrorCaptured } from 'vue'

import { useCapturedErrors } from './captured-errors.ts'

test('capture retains the five newest errors with distinct identities and clears them', () => {
  const { capture, errors, clear } = useCapturedErrors()
  clear()
  for (let i = 0; i < 7; i++) capture(new Error(`Failure ${i}`))
  assert.deepEqual(errors.value.map((error) => error.message), [
    'Failure 6', 'Failure 5', 'Failure 4', 'Failure 3', 'Failure 2',
  ])
  assert.equal(new Set(errors.value.map((error) => error.id)).size, 5)
  assert.match(errors.value[0].stack ?? '', /Failure 6/)
  clear()
  assert.equal(errors.value.length, 0)
})

test('capturing a Vue component error preserves application error reporting', () => {
  const { capture, errors, clear } = useCapturedErrors()
  clear()
  const failure = new Error('Component failed')
  const renderer = createRenderer<object, object>({
    createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
    insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {},
    parentNode: () => null, nextSibling: () => null,
  })
  const Child = defineComponent({ setup() { throw failure } })
  const Parent = defineComponent({
    setup() {
      onErrorCaptured(capture)
      return () => h(Child)
    },
  })
  const reported: unknown[] = []
  const app = renderer.createApp(Parent)
  app.config.errorHandler = (error) => { reported.push(error) }
  app.config.warnHandler = () => {}
  app.mount({})
  assert.deepEqual(reported, [failure])
  assert.equal(errors.value[0].message, failure.message)
  app.unmount()
  clear()
})
