<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { PopoverContent, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { AlertTriangle, ChevronDown, ChevronUp, Copy, X } from '@lucide/vue'

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
const open = ref(false)
const errorExpanded = ref<number | null>(null)

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

function toggleErrorExpanded(id: number) {
  errorExpanded.value = errorExpanded.value === id ? null : id
}

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
  <div
    class="studio-debug-indicator"
    :data-state="open ? 'open' : 'closed'"
    :data-has-errors="hasErrors ? 'true' : 'false'"
  >
    <PopoverRoot v-model:open="open">
      <PopoverContent class="studio-debug-indicator__panel" side="top" align="start" :side-offset="10" aria-label="Studio debug">
        <div class="studio-debug-indicator__header">
          <span class="studio-debug-indicator__title">Studio</span>
          <button
            class="studio-debug-indicator__icon-btn"
            type="button"
            aria-label="Close"
            @click="open = false"
          >
            <X :size="13" :stroke-width="2" aria-hidden="true" />
          </button>
        </div>

        <div v-if="hasErrors" class="studio-debug-indicator__errors">
          <div class="studio-debug-indicator__errors-header">
            <AlertTriangle :size="12" :stroke-width="2" aria-hidden="true" />
            <span>{{ capturedErrors.length }} error{{ capturedErrors.length === 1 ? '' : 's' }} captured</span>
            <button class="studio-debug-indicator__clear" type="button" @click="clearErrors">
              Clear
            </button>
          </div>
          <div
            v-for="err in capturedErrors"
            :key="err.id"
            class="studio-debug-indicator__error"
          >
            <button
              class="studio-debug-indicator__error-toggle"
              type="button"
              :aria-expanded="errorExpanded === err.id"
              @click="toggleErrorExpanded(err.id)"
            >
              <ChevronDown v-if="errorExpanded !== err.id" :size="11" :stroke-width="2.5" aria-hidden="true" />
              <ChevronUp v-else :size="11" :stroke-width="2.5" aria-hidden="true" />
              <span class="studio-debug-indicator__error-msg">{{ err.message }}</span>
            </button>
            <pre v-if="errorExpanded === err.id && err.stack" class="studio-debug-indicator__stack">{{ err.stack }}</pre>
          </div>
        </div>

        <dl class="studio-debug-indicator__list">
          <div class="studio-debug-indicator__row">
            <dt>Route</dt>
            <dd>
              <span class="studio-debug-indicator__mono">{{ routeName }}</span>
              <span class="studio-debug-indicator__path">{{ routePath }}</span>
            </dd>
          </div>
          <div v-if="entity" class="studio-debug-indicator__row">
            <dt>Entity</dt>
            <dd class="studio-debug-indicator__mono">{{ entity }}</dd>
          </div>
          <div v-if="recordName" class="studio-debug-indicator__row">
            <dt>Record</dt>
            <dd class="studio-debug-indicator__mono">{{ recordName }}</dd>
          </div>
          <div class="studio-debug-indicator__row">
            <dt>User</dt>
            <dd>
              <span class="studio-debug-indicator__mono">{{ userEmail }}</span>
              <span class="studio-debug-indicator__path">{{ userRole }}</span>
            </dd>
          </div>
          <div class="studio-debug-indicator__row">
            <dt>Boot</dt>
            <dd class="studio-debug-indicator__mono">{{ bootLabel }}</dd>
          </div>
        </dl>

        <div class="studio-debug-indicator__footer">
          <button class="studio-debug-indicator__btn" type="button" @click="copyBundle">
            <Copy :size="12" :stroke-width="2" aria-hidden="true" />
            Copy bundle
          </button>
          <button
            class="studio-debug-indicator__btn studio-debug-indicator__btn--muted"
            type="button"
            @click="open = false"
          >
            Close
          </button>
        </div>
      </PopoverContent>

      <PopoverTrigger
        class="studio-debug-indicator__trigger"
        type="button"
        aria-label="Open Studio debug indicator"
      >
        <span
          v-if="hasErrors"
          class="studio-debug-indicator__error-dot"
          aria-label="Errors present"
        />
        <span class="studio-debug-indicator__mark" aria-hidden="true">d</span>
      </PopoverTrigger>
    </PopoverRoot>
  </div>
</template>

<style>
.studio-debug-indicator {
  position: fixed;
  left: 16px;
  bottom: 16px;
  z-index: 55;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.studio-debug-indicator__trigger {
  position: relative;
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--studio-border-strong);
  border-radius: 10px;
  background: var(--studio-surface);
  box-shadow: var(--studio-shadow);
  color: var(--studio-text);
  cursor: pointer;
  transition: transform 120ms ease, box-shadow 120ms ease;
}

.studio-debug-indicator__trigger:hover {
  transform: translateY(-1px);
  box-shadow: var(--studio-shadow-sheet);
}

.studio-debug-indicator__trigger:focus-visible {
  outline: 2px solid var(--studio-focus);
  outline-offset: 2px;
}

.studio-debug-indicator__mark {
  font-size: 15px;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
}

.studio-debug-indicator__error-dot {
  position: absolute;
  top: 7px;
  right: 7px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--studio-danger);
  box-shadow: 0 0 0 1.5px var(--studio-surface);
}

.studio-debug-indicator__panel {
  width: min(320px, calc(100vw - 32px));
  max-height: calc(100dvh - 78px);
  overflow-y: auto;
  border: 1px solid var(--studio-border-strong);
  border-radius: 12px;
  background: var(--studio-surface);
  box-shadow: var(--studio-shadow-sheet);
  color: var(--studio-text);
}

.studio-debug-indicator__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border-bottom: 1px solid var(--studio-border);
  padding: 10px 12px;
}

.studio-debug-indicator__title {
  font-size: 12px;
  font-weight: 650;
  letter-spacing: -0.01em;
}

.studio-debug-indicator__icon-btn {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--studio-text-muted);
  cursor: pointer;
}

.studio-debug-indicator__icon-btn:hover {
  background: var(--studio-surface-raised);
  color: var(--studio-text);
}

.studio-debug-indicator__errors {
  border-bottom: 1px solid var(--studio-border);
}

.studio-debug-indicator__errors-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px 6px;
  color: var(--studio-danger);
  font-size: 11px;
  font-weight: 600;
}

.studio-debug-indicator__clear {
  margin-left: auto;
  border: 0;
  background: transparent;
  color: var(--studio-text-muted);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.studio-debug-indicator__clear:hover {
  color: var(--studio-text);
}

.studio-debug-indicator__error {
  border-top: 1px solid var(--studio-border);
}

.studio-debug-indicator__error-toggle {
  display: flex;
  width: 100%;
  align-items: flex-start;
  gap: 6px;
  border: 0;
  background: transparent;
  color: var(--studio-text);
  padding: 7px 12px;
  text-align: left;
  cursor: pointer;
}

.studio-debug-indicator__error-toggle:hover {
  background: var(--studio-surface-raised);
}

.studio-debug-indicator__error-toggle svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--studio-text-muted);
}

.studio-debug-indicator__error-msg {
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.studio-debug-indicator__stack {
  overflow-x: auto;
  margin: 0;
  border-top: 1px solid var(--studio-border);
  background: var(--studio-surface-raised);
  color: var(--studio-text-muted);
  padding: 8px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.studio-debug-indicator__list {
  display: grid;
  margin: 0;
  padding: 4px 0;
}

.studio-debug-indicator__row {
  display: grid;
  grid-template-columns: 60px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  padding: 7px 12px;
}

.studio-debug-indicator__row dt {
  margin: 0;
  color: var(--studio-text-muted);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.35;
}

.studio-debug-indicator__row dd {
  display: grid;
  gap: 2px;
  margin: 0;
  min-width: 0;
  font-size: 12px;
  line-height: 1.35;
}

.studio-debug-indicator__mono {
  overflow: hidden;
  color: var(--studio-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 550;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.studio-debug-indicator__path {
  overflow: hidden;
  color: var(--studio-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.studio-debug-indicator__footer {
  display: flex;
  gap: 6px;
  border-top: 1px solid var(--studio-border);
  padding: 10px 12px;
}

.studio-debug-indicator__btn {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--studio-border-strong);
  border-radius: 7px;
  background: var(--studio-control-bg);
  color: var(--studio-text);
  padding: 0 10px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.studio-debug-indicator__btn:hover {
  background: var(--studio-control-bg-hover);
}

.studio-debug-indicator__btn--muted {
  margin-left: auto;
  border-color: transparent;
  background: transparent;
  color: var(--studio-text-muted);
}

.studio-debug-indicator__btn--muted:hover {
  background: var(--studio-surface-raised);
  color: var(--studio-text);
}

@media (prefers-reduced-motion: reduce) {
  .studio-debug-indicator__trigger {
    transition: none;
  }
}
</style>
