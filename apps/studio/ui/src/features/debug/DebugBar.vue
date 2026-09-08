<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { AlertTriangle, Copy } from '@lucide/vue'

import { routeParam } from '@/router/routes'
import { useAuthStore } from '@/stores/auth.store'
import { useBootStore } from '@/stores/boot.store'
import { useCapturedErrors } from './captured-errors'
import { useToast } from '@/features/toasts/use-toast'

const authStore = useAuthStore()
const bootStore = useBootStore()
const route = useRoute()
const { errors: capturedErrors, clear: clearErrors } = useCapturedErrors()

const toast = useToast()

const hasErrors = computed(() => capturedErrors.value.length > 0)

const routeName = computed(() => String(route.name ?? '—'))
const routePath = computed(() => route.fullPath || route.path || '/')
const entity = computed(() => {
  const value = route.params.entity
  if (typeof value !== 'string' && !Array.isArray(value)) return ''
  return routeParam(value)
})
const recordName = computed(() => {
  const value = route.params.recordName
  if (typeof value !== 'string' && !Array.isArray(value)) return ''
  return routeParam(value)
})
const userEmail = computed(() => authStore.currentUser?.email ?? 'signed out')
const userRole = computed(() => {
  const user = authStore.currentUser
  if (!user) return '—'
  return user.administrator ? 'administrator' : 'user'
})
const bootLabel = computed(() => bootStore.status || 'idle')

function buildDebugBundle(): string {
  return JSON.stringify({
    timestamp: new Date().toISOString(),
    route: { name: routeName.value, path: routePath.value },
    entity: entity.value || null,
    record: recordName.value || null,
    user: { email: userEmail.value, role: userRole.value },
    boot: bootLabel.value,
    errors: capturedErrors.value.map((e) => ({
      message: e.message,
      timestamp: e.timestamp,
      stack: e.stack ?? null,
    })),
  }, null, 2)
}

async function copyBundle() {
  if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
    toast.error('Clipboard is unavailable.')
    return
  }

  try {
    await navigator.clipboard.writeText(buildDebugBundle())
    toast.success('Debug bundle copied.')
  } catch {
    toast.error('Could not copy debug bundle.')
  }
}
</script>

<template>
  <footer class="studio-debug-bar" aria-label="Studio debug">
    <div class="studio-debug-bar__status">
      <strong>dygo</strong>
      <dl>
        <div><dt>Route</dt><dd>{{ routeName }} · {{ routePath }}</dd></div>
        <div v-if="entity"><dt>Entity</dt><dd>{{ entity }}</dd></div>
        <div v-if="recordName"><dt>Record</dt><dd>{{ recordName }}</dd></div>
        <div><dt>User</dt><dd>{{ userEmail }} · {{ userRole }}</dd></div>
        <div><dt>Boot</dt><dd>{{ bootLabel }}</dd></div>
      </dl>
      <button type="button" @click="copyBundle"><Copy :size="12" aria-hidden="true" />Copy bundle</button>
    </div>
    <details v-if="hasErrors" class="studio-debug-bar__errors">
      <summary><AlertTriangle :size="12" aria-hidden="true" />{{ capturedErrors.length }} error{{ capturedErrors.length === 1 ? '' : 's' }}</summary>
      <div class="studio-debug-bar__error-list">
        <button type="button" @click="clearErrors">Clear errors</button>
        <details v-for="error in capturedErrors" :key="error.id">
          <summary>{{ error.message }}</summary>
          <pre v-if="error.stack">{{ error.stack }}</pre>
        </details>
      </div>
    </details>
  </footer>
</template>

<style scoped>
.studio-debug-bar {
  flex: none;
  border-top: 1px solid var(--studio-border);
  background: var(--studio-surface);
  color: var(--studio-text-muted);
  font-size: 11px;
}

.studio-debug-bar__status,
.studio-debug-bar dl,
.studio-debug-bar dl > div,
.studio-debug-bar button {
  display: flex;
  align-items: center;
  gap: 8px;
}

.studio-debug-bar__status { min-height: 30px; padding: 0 12px; }
.studio-debug-bar strong { color: var(--studio-text); }
.studio-debug-bar dl {
  flex: 1;
  min-width: 0;
  margin: 0;
  overflow-x: auto;
  gap: 20px;
  white-space: nowrap;
}

.studio-debug-bar dd {
  margin: 0;
  color: var(--studio-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.studio-debug-bar button {
  flex: none;
  min-height: 26px;
  padding: 0 6px;
  border: 0;
  border-radius: var(--studio-radius-control);
  background: transparent;
  color: var(--studio-text);
  font: inherit;
  cursor: pointer;
  white-space: nowrap;
}

.studio-debug-bar button:hover { background: var(--studio-surface-raised); }
.studio-debug-bar :is(button, summary):focus-visible { outline: 2px solid var(--studio-focus); outline-offset: -2px; }
.studio-debug-bar summary { padding: 6px 12px; cursor: pointer; overflow-wrap: anywhere; }
.studio-debug-bar summary svg { margin-right: 6px; vertical-align: middle; }
.studio-debug-bar__errors { border-top: 1px solid var(--studio-border); color: var(--studio-danger); }
.studio-debug-bar__error-list { max-height: 25dvh; overflow: auto; padding: 0 12px 8px; }
.studio-debug-bar__error-list summary { padding-inline: 0; }
.studio-debug-bar pre { margin: 0; color: var(--studio-text-muted); white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
