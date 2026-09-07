import test from 'node:test'
import assert from 'node:assert/strict'
import { secretSubmitValue } from './secret-input.ts'

test('secret input preserves existing values and clears only explicitly', () => {
  for (const value of [undefined, '']) assert.deepEqual(secretSubmitValue(value, true, true), { skip: true })
  assert.deepEqual(secretSubmitValue(null, false, true), { value: null })
  assert.deepEqual(secretSubmitValue(' replacement ', true, true), { value: ' replacement ' })
  assert.ok(secretSubmitValue(undefined, true, false).error)
  assert.ok(secretSubmitValue(null, true, true).error)
})
