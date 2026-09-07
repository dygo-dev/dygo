import { computed, onScopeDispose, shallowRef, watchEffect, type ComputedRef } from 'vue'

export type PageCommand = {
  id: string
  label: string
  detail?: string
  keywords?: string
  run: () => void | Promise<void>
  disabled?: boolean
}

const active = shallowRef<{ owner: symbol, commands: PageCommand[] } | null>(null)
export const pageCommands = computed(() => active.value?.commands ?? [])

export function usePageCommands(commands: ComputedRef<PageCommand[]>) {
  const owner = Symbol('page commands')
  watchEffect(() => { active.value = { owner, commands: commands.value } })
  onScopeDispose(() => {
    if (active.value?.owner === owner) active.value = null
  })
}
