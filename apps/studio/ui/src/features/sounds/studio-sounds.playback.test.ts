import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

test('playConfiguredSound does not enforce a playback duration cap', () => {
  const source = readFileSync(new URL('./studio-sounds.ts', import.meta.url), 'utf8')

  assert.doesNotMatch(source, /setTimeout/)
  assert.doesNotMatch(source, /playbackLimit/)
  assert.doesNotMatch(source, /studioSoundPlaybackSeconds/)
})
