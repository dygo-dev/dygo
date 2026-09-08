import type { StudioDialogType } from '@/features/dialogs/dialogs.store'
import type { EntityActionDefinition } from '@/features/metadata/metadata.api'

import type { RecordData } from './records.api'

export type EntityActionConfirm = {
  title: string
  content: string
  type: StudioDialogType
}

export function recordEntityActions(actions: EntityActionDefinition[] | undefined): EntityActionDefinition[] {
  return (actions ?? []).filter((action): action is EntityActionDefinition => Boolean(action) && action.selection === 'record')
}

export function entityActionDisabledReason(
  entityKey: string,
  actionName: string,
  record: RecordData | null | undefined,
): string | undefined {
  if (!record) {
    return 'Record unavailable'
  }
  if (entityKey !== 'job-execution') {
    return undefined
  }

  const status = typeof record.status === 'string' ? record.status : ''
  if (actionName === 'cancel' && status !== 'queued') {
    return 'Only queued Job Executions can be cancelled'
  }
  if (actionName === 'retry' && status !== 'failed') {
    return 'Only failed Job Executions can be retried'
  }
  return undefined
}

export function entityActionConfirm(action: EntityActionDefinition): EntityActionConfirm | null {
  const content = action.confirm?.trim() ?? ''
  if (!content) {
    return null
  }

  return {
    title: `${action.label}?`,
    content,
    type: action.danger ? 'danger' : 'warning',
  }
}
