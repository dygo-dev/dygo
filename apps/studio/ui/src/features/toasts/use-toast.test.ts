import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

test('useToast plays the error sound for danger toasts', () => {
  const source = readFileSync(new URL('./use-toast.ts', import.meta.url), 'utf8')

  assert.match(source, /studioSounds\.error\(\)/)
})
