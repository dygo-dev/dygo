import { computed, onScopeDispose, shallowRef, watch, type ComputedRef } from 'vue'
import { commandBinding, executeCommand, type StudioCommand } from './shortcuts.ts'

export type PageCommand = StudioCommand

const pages = shallowRef<{ owner: symbol, commands: PageCommand[] }[]>([])
export const globalCommands = shallowRef<PageCommand[]>([])
export const pageCommands = computed(() => pages.value.at(-1)?.commands ?? [])
export const studioCommands = computed(() => [...globalCommands.value, ...pageCommands.value].map(commandBinding))
export function runStudioCommand(id: string) {
  const command = studioCommands.value.find(command => command.id === id)
  if (command) return executeCommand(command)
}

export function usePageCommands(commands: ComputedRef<PageCommand[]>) {
  const owner = Symbol('page commands')
  pages.value = [...pages.value, { owner, commands: [] }]
  watch(commands, value => { pages.value = pages.value.map(page => page.owner === owner ? { owner, commands: value.map(commandBinding) } : page) }, { immediate: true })
  onScopeDispose(() => {
    pages.value = pages.value.filter(page => page.owner !== owner)
  })
}
