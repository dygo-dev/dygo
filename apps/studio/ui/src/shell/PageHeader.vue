<script setup lang="ts">
import { computed, useSlots } from 'vue'
import { ariaShortcut, shortcutLabel } from '@/features/commands/shortcuts'
import { Pin, PinOff } from '@lucide/vue'

import Badge from '@/design/atoms/Badge.vue'
import Button from '@/design/atoms/Button.vue'
import Breadcrumbs from './Breadcrumbs.vue'
import type { PageHeaderAction } from './types'
import { pinnedItemID, type PinnedItem } from '@/features/pinned/pinned'
import { usePreferencesStore } from '@/features/preferences/preferences.store'
import { useNavigationStore } from '@/stores/navigation.store'

const props = withDefaults(defineProps<{
  title?: string
  titleId?: string
  showBreadcrumbs?: boolean
  showTitle?: boolean
  showActions?: boolean
  system?: boolean
  actions?: PageHeaderAction[]
  pinTarget?: PinnedItem | null
}>(), {
  showBreadcrumbs: true,
  showTitle: true,
  showActions: true,
  system: false,
  actions: () => [],
  pinTarget: null,
})

const navigation = useNavigationStore()
const preferences = usePreferencesStore()
const navigationStoreReady = computed(() => preferences.ready)
const pinned = computed(() => props.pinTarget
  ? navigation.pinnedItems.some(item => pinnedItemID(item) === pinnedItemID(props.pinTarget as PinnedItem))
  : false)

const slots = useSlots()

const hasBreadcrumbs = computed(() => props.showBreadcrumbs)
const hasTitle = computed(() => props.showTitle && Boolean(props.title || slots.title))
const hasMain = computed(() => hasBreadcrumbs.value || hasTitle.value)
const hasActions = computed(() => props.showActions && (props.actions.length > 0 || Boolean(slots.actions)))

function runAction(action: PageHeaderAction) {
  if (action.disabled || action.loading) {
    return
  }

  action.onSelect?.()
}
</script>

<template>
  <header
    class="studio-page-header"
    :class="{ 'studio-page-header--with-actions': hasActions }"
  >
    <div v-if="hasMain" class="studio-page-header__main">
      <div v-if="hasBreadcrumbs" class="studio-page-header__breadcrumb-row">
        <Breadcrumbs class="studio-page-header__breadcrumbs" />
        <Badge v-if="props.system" variant="danger">System</Badge>
        <button
          v-if="props.pinTarget"
          class="studio-page-header__pin"
          type="button"
          :disabled="!navigationStoreReady"
          :aria-label="pinned ? 'Unpin from sidebar' : 'Pin to sidebar'"
          :title="pinned ? 'Unpin from sidebar' : 'Pin to sidebar'"
          :aria-pressed="pinned"
          @click="navigation.togglePin(props.pinTarget)"
        >
          <PinOff v-if="pinned" :size="15" aria-hidden="true" />
          <Pin v-else :size="15" aria-hidden="true" />
        </button>
      </div>
      <h1 v-if="hasTitle" :id="props.titleId" class="studio-page-header__title">
        <slot name="title">{{ props.title }}</slot>
      </h1>
    </div>

    <div v-if="hasActions" class="studio-page-header__actions">
      <slot name="actions">
        <Button
          v-for="action in props.actions"
          :key="action.label"
          type="button"
          :variant="action.variant ?? 'secondary'"
          :disabled="action.disabled"
          :loading="action.loading"
          :aria-keyshortcuts="ariaShortcut(action.shortcut)"
          :title="action.shortcut ? `${action.label} (${shortcutLabel(action.shortcut)})` : undefined"
          size="sm"
          @click="runAction(action)"
        >
          <component
            :is="action.icon"
            v-if="action.icon"
            class="studio-page-header__action-icon"
            :size="15"
            :stroke-width="1.8"
            aria-hidden="true"
          />
          {{ action.label }}
        </Button>
      </slot>
    </div>
  </header>
</template>

<style scoped>
.studio-page-header {
  display: grid;
  min-width: 0;
  gap: 8px;
  margin: calc(var(--studio-page-padding) * -1) calc(var(--studio-page-padding) * -1) 0;
  border-bottom: 1px solid var(--studio-border);
  padding: 8px var(--studio-page-padding);
  min-height: 49px;
}

.studio-page-header--with-actions {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}

.studio-page-header__main {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.studio-page-header__breadcrumb-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.studio-page-header__breadcrumbs {
  min-width: 0;
}

.studio-page-header__pin {
  display: inline-flex;
  width: var(--studio-control-height-sm);
  height: var(--studio-control-height-sm);
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--studio-radius-control);
  background: transparent;
  color: var(--studio-text-muted);
}

.studio-page-header__pin:hover:not(:disabled) {
  background: var(--studio-surface-raised);
  color: var(--studio-text);
}

.studio-page-header__pin[aria-pressed='true'] {
  color: var(--studio-accent);
}

.studio-page-header__pin:focus-visible {
  outline: 2px solid var(--studio-focus);
  outline-offset: 2px;
}

.studio-page-header__title {
  margin: 0;
  color: var(--studio-text);
  font-size: 20px;
  font-weight: 500;
  letter-spacing: 0;
  line-height: 1.16;
}

.studio-page-header__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.studio-page-header__action-icon {
  flex: 0 0 auto;
}

@media (max-width: 720px) {
  .studio-page-header--with-actions {
    grid-template-columns: minmax(0, 1fr);
  }

  .studio-page-header__actions {
    justify-content: flex-start;
  }
}
</style>
