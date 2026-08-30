import test from 'node:test'
import assert from 'node:assert/strict'

import { normalizePageClaimPath, pageRouteName } from '../../router/routes.ts'

test('Page claims register only exact internal paths', () => {
  assert.equal(normalizePageClaimPath('/workspace/'), '/workspace')
  assert.equal(normalizePageClaimPath('/'), '/')
  assert.equal(normalizePageClaimPath('//outside.example'), null)
  assert.equal(normalizePageClaimPath('/:entity'), null)
  assert.equal(normalizePageClaimPath('/search?query=one'), null)
  assert.equal(normalizePageClaimPath('/report/*'), null)
})

test('Page route names retain app-scoped identity', () => {
  assert.equal(pageRouteName('studio', 'home'), 'page:studio:home')
  assert.notEqual(pageRouteName('sales', 'home'), pageRouteName('studio', 'home'))
})
