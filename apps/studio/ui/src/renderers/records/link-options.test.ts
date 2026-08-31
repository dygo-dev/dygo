import test from 'node:test'
import assert from 'node:assert/strict'

import { resolveLinkFilterValue, resolveLinkFilters } from './link-options.ts'

test('resolveLinkFilterValue resolves current-form tokens', () => {
  assert.equal(resolveLinkFilterValue('$department', { department: 'engineering' }), 'engineering')
  assert.equal(resolveLinkFilterValue('${department}', { department: 'engineering' }), 'engineering')
  assert.equal(resolveLinkFilterValue('Active', {}), 'Active')
})

test('resolveLinkFilters omits unresolved dependent filters', () => {
  assert.deepEqual(resolveLinkFilters([
    { field: 'department', operator: 'eq', value: '$department' },
    { field: 'status', operator: 'eq', value: 'Active' },
    { field: 'archived', operator: 'empty' },
  ], {}), [
    { field: 'status', operator: 'eq', value: 'Active' },
    { field: 'archived', operator: 'empty' },
  ])
})
