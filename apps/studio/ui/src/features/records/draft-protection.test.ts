import assert from 'node:assert/strict'
import test from 'node:test'
import { draftProtection } from './draft-protection.ts'

test('reset and navigation share one confirmation without changing or saving the draft', async () => {
  let dirty = true
  let busy = false
  let prompts = 0
  let decide!: (value: boolean) => void
  const confirm = draftProtection(() => dirty, () => busy, () => { prompts++; return new Promise(resolve => { decide = resolve }) })
  const reset = confirm()
  const navigation = confirm()
  assert.equal(prompts, 1)
  decide(false)
  assert.equal(await reset, false)
  assert.equal(await navigation, false)
  assert.equal(dirty, true)
  const retry = confirm(); decide(true)
  assert.equal(await retry, true)
  assert.equal(dirty, true, 'caller resets only after its confirmed action')
  busy = true
  assert.equal(await confirm(), false)
  busy = false; dirty = false
  assert.equal(await confirm(), true)
  assert.equal(prompts, 2)
})
