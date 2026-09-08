import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizePinnedItems, pinnedItemID } from './pinned.ts'

test('normalizes supported pins and removes duplicate identities', () => {
  const pins = normalizePinnedItems([
    { type: 'entity', app: 'sales', entity: 'customer', label: 'Customers', path: '/customers' },
    { type: 'entity', app: 'sales', entity: 'customer', label: 'Old label', path: '/old' },
    { type: 'record', app: 'sales', entity: 'customer', record: 'CUS-1', label: 'Customer / CUS-1', path: '/customers/CUS-1' },
    { type: 'page', app: 'studio', page: 'home', label: 'Home', path: '/' },
    { type: 'record', app: 'sales', entity: 'customer', label: 'Missing record', path: '/customers' },
  ])

  assert.equal(pins.length, 3)
  assert.equal(pins[0]?.label, 'Customers')
  assert.equal(pinnedItemID(pins[1]!), 'record:sales:customer:CUS-1')
})
