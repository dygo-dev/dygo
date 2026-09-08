import assert from 'node:assert/strict'
import test from 'node:test'

import {
  entityActionConfirm,
  entityActionDisabledReason,
  recordEntityActions,
} from './entity-actions.ts'

test('recordEntityActions keeps only record selection actions', () => {
  const actions = recordEntityActions([
    { name: 'cancel', label: 'Cancel', selection: 'record' },
    { name: 'bulk-close', label: 'Close', selection: 'selection' },
    { name: 'reindex', label: 'Reindex', selection: 'collection' },
  ])

  assert.deepEqual(actions.map((action) => action.name), ['cancel'])
})

test('entityActionDisabledReason gates Job Execution cancel and retry', () => {
  assert.equal(entityActionDisabledReason('job-execution', 'cancel', null), 'Record unavailable')
  assert.equal(entityActionDisabledReason('job-execution', 'cancel', { status: 'queued' }), undefined)
  assert.equal(
    entityActionDisabledReason('job-execution', 'cancel', { status: 'running' }),
    'Only queued Job Executions can be cancelled',
  )
  assert.equal(entityActionDisabledReason('job-execution', 'retry', { status: 'failed' }), undefined)
  assert.equal(
    entityActionDisabledReason('job-execution', 'retry', { status: 'queued' }),
    'Only failed Job Executions can be retried',
  )
  assert.equal(entityActionDisabledReason('lead', 'cancel', { status: 'open' }), undefined)
})

test('entityActionConfirm uses action metadata', () => {
  assert.equal(entityActionConfirm({ name: 'archive', label: 'Archive', selection: 'record' }), null)
  assert.deepEqual(
    entityActionConfirm({
      name: 'cancel',
      label: 'Cancel',
      selection: 'record',
      confirm: 'This queued Job Execution will not run.',
      danger: true,
    }),
    {
      title: 'Cancel?',
      content: 'This queued Job Execution will not run.',
      type: 'danger',
    },
  )
  assert.deepEqual(
    entityActionConfirm({
      name: 'retry',
      label: 'Retry',
      selection: 'record',
      confirm: 'dygo will queue a new Job Execution with the same payload.',
    }),
    {
      title: 'Retry?',
      content: 'dygo will queue a new Job Execution with the same payload.',
      type: 'warning',
    },
  )
})
