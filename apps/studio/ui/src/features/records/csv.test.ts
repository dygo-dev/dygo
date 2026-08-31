import test from 'node:test'
import assert from 'node:assert/strict'

import { recordsToCSV } from './csv.ts'

test('recordsToCSV escapes cells and preserves column order', () => {
  assert.equal(recordsToCSV([{ name: 'A, B', status: 'Present\nToday' }], [
    { key: 'name', label: 'Name' },
    { key: 'status', label: 'Status' },
  ]), 'Name,Status\r\n"A, B","Present\nToday"\r\n')
})
