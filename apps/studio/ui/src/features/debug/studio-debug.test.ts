import assert from 'node:assert/strict'
import test from 'node:test'

import { isStudioDebugBarAvailable } from './studio-debug.ts'

test('debug indicator is available only in development builds', () => {
  assert.equal(isStudioDebugBarAvailable(true), true)
  assert.equal(isStudioDebugBarAvailable(false), false)
})
