import assert from 'node:assert/strict'
import test from 'node:test'
import { searchTree, treeRecord } from './tree.ts'
import type { TreeNode } from './tree.api'

const match = (name: string, parent: string | null, pathUnavailable = false): TreeNode => ({ record: { name, parent }, hasChildren: false, matched: true, pathUnavailable })

test('tree search shares ancestor context without duplicating matches', () => {
  const result = searchTree([match('one', 'root'), match('two', 'root'), match('root', null)], [{ name: 'root', parent: null }], 'parent')
  assert.equal(result.length, 1)
  assert.equal(result[0]!.contextOnly, false)
  assert.deepEqual(result[0]!.children?.map((item) => item.key), ['record:one', 'record:two'])
})

test('unavailable paths stay explicit and labels fall back to Record names', () => {
  const result = searchTree([match('visible', 'hidden', true)], [], 'parent')
  assert.equal(result[0]!.pathUnavailable, true)
  assert.equal(result[0]!.key, 'record:visible')
  assert.equal(result[0]!.children, undefined)
  assert.equal(treeRecord(match('ID', null), 'title').label, 'ID')
  assert.equal(treeRecord({ ...match('ID', null), record: { name: 'ID', title: 'Title' } }, 'title').label, 'Title')
})
