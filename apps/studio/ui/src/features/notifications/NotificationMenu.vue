<script setup lang="ts">
import { computed } from 'vue'
import { Bell } from '@lucide/vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth.store'
import { useNotificationCountQuery, useNotificationsQuery, useOpenNotification } from './notifications.query'

const router = useRouter()
const authStore = useAuthStore()
const enabled = computed(() => Boolean(authStore.currentUser))
const notifications = useNotificationsQuery(enabled)
const count = useNotificationCountQuery(enabled)
const openNotification = useOpenNotification()

const items = computed(() => notifications.data.value ?? [])
const unreadCount = computed(() => count.data.value ?? 0)
const badge = computed(() => unreadCount.value > 99 ? '99+' : String(unreadCount.value))

async function refresh(open: boolean) {
  if (!open) return
  await Promise.all([notifications.refetch(), count.refetch()])
}

async function selectNotification(id: number) {
  try {
    const deepLink = await openNotification.mutateAsync(id)
    await router.push(deepLink)
  } catch {
    // The menu keeps its error state. Global API messaging handles the failure detail.
  }
}
</script>

<template>
  <DropdownMenuRoot @update:open="refresh">
    <DropdownMenuTrigger as-child>
      <button class="studio-notifications__trigger" type="button" aria-label="Notifications">
        <Bell :size="16" :stroke-width="1.8" aria-hidden="true" />
        <span v-if="unreadCount > 0" class="studio-notifications__badge" aria-label="Unread notifications">{{ badge }}</span>
      </button>
    </DropdownMenuTrigger>
    <DropdownMenuPortal>
      <DropdownMenuContent class="studio-notifications__content" align="end" :side-offset="6">
        <DropdownMenuLabel class="studio-notifications__label">
          <span>Notifications</span>
          <span v-if="unreadCount > 0">{{ unreadCount }} unread</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator class="studio-notifications__separator" />

        <p v-if="notifications.isPending.value" class="studio-notifications__state">Loading notifications</p>
        <button v-else-if="notifications.error.value" class="studio-notifications__retry" type="button" @click="notifications.refetch()">
          Could not load notifications. Try again.
        </button>
        <p v-else-if="items.length === 0" class="studio-notifications__state">All caught up.</p>
        <template v-else>
          <DropdownMenuItem
            v-for="item in items"
            :key="item.id"
            class="studio-notifications__item"
            :disabled="openNotification.isPending.value"
            @select="selectNotification(item.id)"
          >
            <strong>{{ item.title }}</strong>
            <span>{{ item.message }}</span>
            <time :datetime="item['created-at']">{{ new Date(item['created-at']).toLocaleString() }}</time>
          </DropdownMenuItem>
        </template>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>

<style>
.studio-notifications__trigger {
  position: relative;
  display: inline-flex;
  width: var(--studio-control-height-xs);
  height: var(--studio-control-height-xs);
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--studio-radius-control);
  background: transparent;
  color: var(--studio-text-muted);
}

.studio-notifications__trigger:hover {
  background: var(--studio-surface-raised);
  color: var(--studio-text);
}

.studio-notifications__trigger:focus-visible {
  outline: 2px solid var(--studio-focus);
  outline-offset: 2px;
}

.studio-notifications__badge {
  position: absolute;
  top: -4px;
  right: -6px;
  min-width: 16px;
  height: 16px;
  border: 2px solid var(--studio-bg);
  border-radius: 999px;
  background: var(--studio-danger);
  color: var(--studio-danger-contrast);
  font-size: 9px;
  font-weight: 700;
  line-height: 12px;
  padding-inline: 3px;
}

.studio-notifications__content {
  z-index: 1000;
  width: min(360px, calc(100vw - 24px));
  max-height: min(480px, calc(100vh - 40px));
  overflow-y: auto;
  border: 1px solid var(--studio-border);
  border-radius: var(--studio-radius-panel);
  background: var(--studio-surface);
  box-shadow: var(--studio-shadow);
  padding: 5px;
}

.studio-notifications__label {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 8px;
  color: var(--studio-text);
  font-size: 12px;
  font-weight: 700;
}

.studio-notifications__label span:last-child {
  color: var(--studio-text-subtle);
  font-weight: 600;
}

.studio-notifications__separator {
  height: 1px;
  margin: 3px -5px 5px;
  background: var(--studio-border);
}

.studio-notifications__item {
  display: grid;
  gap: 3px;
  min-height: 64px;
  border-radius: 6px;
  outline: none;
  padding: 9px 10px;
  user-select: none;
}

.studio-notifications__item[data-highlighted] {
  background: var(--studio-surface-raised);
}

.studio-notifications__item strong {
  color: var(--studio-text);
  font-size: 13px;
}

.studio-notifications__item span {
  display: -webkit-box;
  overflow: hidden;
  color: var(--studio-text-muted);
  font-size: 12px;
  line-height: 1.4;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.studio-notifications__item time {
  color: var(--studio-text-subtle);
  font-size: 10px;
}

.studio-notifications__state,
.studio-notifications__retry {
  width: 100%;
  margin: 0;
  border: 0;
  background: transparent;
  color: var(--studio-text-subtle);
  font-size: 12px;
  text-align: left;
  padding: 18px 10px;
}

.studio-notifications__retry {
  color: var(--studio-text-muted);
}
</style>
