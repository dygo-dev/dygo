<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'reka-ui'
import { studioCommands } from '@/features/commands/context'
import { shortcutLabel } from '@/features/commands/shortcuts'
import { useNavigationStore } from '@/stores/navigation.store'

const navigation = useNavigationStore()
const search = ref('')
let returnFocus: HTMLElement | null = null
function restoreFocus(event: Event) { event.preventDefault(); returnFocus?.focus() }
watch(() => navigation.shortcutsOpen, open => {
  if (open) { search.value = ''; returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null }
}, { flush: 'sync' })
const groups = computed(() => {
  const result = new Map<string, typeof studioCommands.value>()
  for (const command of studioCommands.value) {
    if (!`${command.label} ${command.disabledReason ?? ''} ${shortcutLabel(command.shortcut)}`.toLowerCase().includes(search.value.trim().toLowerCase())) continue
    const group = command.group ?? 'This page'
    result.set(group, [...(result.get(group) ?? []), command])
  }
  return [...result]
})
</script>

<template>
  <DialogRoot v-model:open="navigation.shortcutsOpen">
    <DialogPortal>
      <DialogOverlay class="studio-command-menu__overlay" />
      <DialogContent class="studio-command-menu__dialog" aria-describedby="shortcut-description" @close-auto-focus="restoreFocus">
        <div class="shortcut-heading"><DialogTitle>Keyboard shortcuts</DialogTitle><DialogClose aria-label="Close keyboard shortcuts">×</DialogClose></div>
        <p id="shortcut-description" class="sr-only">Studio and current-page commands. Commands without a key are available from the command palette.</p>
        <input v-model="search" class="studio-command-menu__input-wrap studio-command-menu__input" aria-label="Search keyboard shortcuts" placeholder="Search commands">
        <div class="studio-command-menu__list">
          <section v-for="[group, commands] in groups" :key="group">
            <h3 class="studio-command-menu__group-label">{{ group }}</h3>
            <div v-for="command in commands" :key="command.id" class="shortcut-row">
              <span>{{ command.label }}<small v-if="command.disabledReason">{{ command.disabledReason }}</small></span>
              <kbd>{{ shortcutLabel(command.shortcut) || 'Command palette' }}</kbd>
            </div>
          </section>
          <p v-if="!groups.length" role="status">No matching commands</p>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.shortcut-heading, .shortcut-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 16px; }
.shortcut-heading { border-bottom: 1px solid var(--studio-border); }
.shortcut-heading h2 { font-size: 16px; margin: 0; }
.shortcut-heading button { background: transparent; color: inherit; border: 0; font-size: 22px; }
.shortcut-row { font-size: 13px; }
.shortcut-row small { display: block; color: var(--studio-text-muted); }
.shortcut-row kbd { color: var(--studio-text-muted); white-space: nowrap; }
</style>
