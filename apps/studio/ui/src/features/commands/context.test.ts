import test from 'node:test'
import assert from 'node:assert/strict'
import { computed, effectScope, nextTick, ref } from 'vue'
import { pageCommands, usePageCommands } from './context.ts'

test('page commands update and dispose without clearing a newer page', async () => {
  const first = effectScope()
  const second = effectScope()
  const label = ref('Clear filters')
  let called = false
  first.run(() => usePageCommands(computed(() => [{ id: 'clear', label: label.value, run: () => { called = true } }])))
  label.value = 'Clear all'
  await nextTick()
  assert.equal(pageCommands.value[0].label, 'Clear all')
  pageCommands.value[0].run()
  assert.equal(called, true)
  second.run(() => usePageCommands(computed(() => [{ id: 'next', label: 'Next page', run() {} }])))
  first.stop()
  assert.equal(pageCommands.value[0].id, 'next')
  second.stop()
  assert.deepEqual(pageCommands.value, [])
})
