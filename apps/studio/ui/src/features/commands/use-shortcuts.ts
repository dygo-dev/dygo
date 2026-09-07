import { onMounted, onScopeDispose, watch } from 'vue'
import { studioCommands } from './context'
import { bindingConflicts, executeCommand, matchShortcut, unavailable } from './shortcuts'

export function useShortcuts() {
  watch(studioCommands, commands => {
    const conflicts = bindingConflicts(commands)
    if (!conflicts.length) return
    const message = `Ambiguous Studio shortcuts: ${conflicts.map(([key, ids]) => `${key}: ${ids.join(', ')}`).join('; ')}`
    if (import.meta.env.DEV) throw new Error(message)
    console.error(message)
  }, { immediate: true })
  function keydown(event: KeyboardEvent) {
    const target = event.target instanceof Element ? event.target : null
    const editable = !!target?.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"]')
    const overlay = !!target?.closest('[role="dialog"], [role="alertdialog"], [role="menu"], [data-reka-popper-content-wrapper]')
      || !!document.querySelector('[role="dialog"][data-state="open"], [role="alertdialog"], [role="menu"][data-state="open"], [data-reka-popper-content-wrapper] [data-state="open"]')
    // The palette may close itself with Mod+K, but cannot invoke background actions.
    const inPalette = !!target?.closest('[data-studio-command-menu]')
    const commands = inPalette ? studioCommands.value.filter(command => command.id === 'app:palette') : studioCommands.value
    const command = matchShortcut(event, commands, editable, overlay && !inPalette)
    if (!command && overlay) {
      // Reserve browser Save Page without submitting behind an overlay.
      if (matchShortcut(event, studioCommands.value, editable, false)?.consumeDisabled) event.preventDefault()
      return
    }
    if (!command || (unavailable(command) && !command.consumeDisabled)) return
    event.preventDefault()
    void executeCommand(command)
  }
  onMounted(() => window.addEventListener('keydown', keydown))
  onScopeDispose(() => window.removeEventListener('keydown', keydown))
}
