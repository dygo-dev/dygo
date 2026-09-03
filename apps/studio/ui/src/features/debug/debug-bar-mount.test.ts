import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

test('App mounts the debug bar on authenticated shell routes', () => {
  const source = readFileSync(new URL('../../app/App.vue', import.meta.url), 'utf8')

  assert.match(source, /import \{ DebugBar \} from '@\/features\/debug'/)
  assert.match(source, /<DebugBar v-if="usesShell" \/>/)
})
