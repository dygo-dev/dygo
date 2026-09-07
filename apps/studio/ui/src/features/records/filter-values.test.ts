import assert from 'node:assert/strict'
import test from 'node:test'
import { filterDisplayValue, filterWireValue, validateFilters } from './filter-values.ts'
import type { MetadataField } from '../metadata/metadata.api'

const field = (type: string, options?: unknown) => ({ name: 'value', label: 'Value', type, options, listable: true, filter: { operators: [{ key: 'eq', arity: 'one' }, { key: 'between', arity: 'range' }, { key: 'empty', arity: 'none' }] } }) as MetadataField

test('saved predicates reject stale fields, operators, options and incomplete ranges', () => {
  for (const options of [['a', 'b'], { values: ['a', 'b'] }]) {
    assert.equal(validateFilters([{ field: 'value', operator: 'eq', value: 'a' }], [field('select', options)]), '')
    assert.ok(validateFilters([{ field: 'value', operator: 'eq', value: 'c' }], [field('select', options)]))
  }
  for (const [type, value] of [['int', '1.5'], ['decimal', '1,000'], ['boolean', 'yes'], ['date', 'not-a-date']]) {
    assert.ok(validateFilters([{ field: 'value', operator: 'eq', value }], [field(type!)]))
  }
  assert.ok(validateFilters([{ field: 'retired', operator: 'eq', value: '1' }], [field('int')]))
  assert.ok(validateFilters([{ field: 'value', operator: 'retired', value: '1' }], [field('int')]))
  assert.ok(validateFilters([{ field: 'value', operator: 'between', value: '1..' }], [field('int')]))
  assert.equal(validateFilters([{ field: 'value', operator: 'between', value: '1..20' }], [field('int')]), '')
  assert.equal(validateFilters([{ field: 'value', operator: 'empty' }], [field('int')]), '')
  assert.equal(validateFilters([{ field: 'name', operator: 'contains', value: 'INV' }], []), '')
})

test('datetime editor preserves the instant through local input conversion', () => {
  const instant = '2026-09-08T08:12:30.000Z'
  assert.equal(filterWireValue(filterDisplayValue(instant, 'datetime'), 'datetime'), instant)
  assert.equal(filterWireValue('9007199254740993', 'bigint'), '9007199254740993')
})
