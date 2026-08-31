import { useMutation } from '@tanstack/vue-query'

import { executeRecordAction } from './records.api'

export function useExecuteRecordActionMutation() {
  return useMutation({
    mutationFn: ({ entity, action, records, input }: {
      entity: string
      action: string
      records: number[]
      input?: Record<string, unknown>
    }) => executeRecordAction(entity, action, records, input),
  })
}
