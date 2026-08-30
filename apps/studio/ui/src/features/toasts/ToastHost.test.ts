import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

test('keeps the toast live region mounted when no toasts are visible', () => {
  const source = readFileSync(new URL('./ToastHost.vue', import.meta.url), 'utf8')

  assert.match(source, /class="studio-toast-host" aria-live="polite"/)
  assert.doesNotMatch(source, /v-if="visibleToasts\.length"/)
})
