<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { onClickOutside } from '@vueuse/core'
import { AlertTriangle, ChevronDown, ChevronUp, Copy, X } from '@lucide/vue'

import { routeParam } from '@/router/routes'
import { useAuthStore } from '@/stores/auth.store'
import { useBootStore } from '@/stores/boot.store'
import { useCapturedErrors } from './captured-errors'
import { isStudioDebugBarAvailable } from './studio-debug'

const authStore = useAuthStore()
const bootStore = useBootStore()
const route = useRoute()
const { errors: capturedErrors, clear: clearErrors } = useCapturedErrors()

const available = isStudioDebugBarAvailable()
const open = ref(false)
const errorExpanded = ref<number | null>(null)
const rootRef = ref<HTMLElement | null>(null)
const copyLabel = ref('Copy bundle')

onClickOutside(rootRef, () => {
  open.value = false
})

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

function toggleOpen() {
  open.value = !open.value
}

function hideIndicator() {
  open.value = false
}

function toggleErrorExpanded(index: number) {
  errorExpanded.value = errorExpanded.value === index ? null : index
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
  if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) return

  try {
    await navigator.clipboard.writeText(buildDebugBundle())
    copyLabel.value = 'Copied!'
    window.setTimeout(() => { copyLabel.value = 'Copy bundle' }, 1400)
  } catch {
    copyLabel.value = 'Copy bundle'
  }
}
</script>

<template>
  <div
    v-if="available"
    ref="rootRef"
    class="studio-debug-indicator"
    :data-state="open ? 'open' : 'closed'"
    :data-has-errors="hasErrors ? 'true' : 'false'"
  >
    <div v-if="open" class="studio-debug-indicator__panel" role="dialog" aria-label="Studio debug">
      <!-- Header -->
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

      <!-- Errors section -->
      <div v-if="hasErrors" class="studio-debug-indicator__errors">
        <div class="studio-debug-indicator__errors-header">
          <AlertTriangle :size="12" :stroke-width="2" aria-hidden="true" />
          <span>{{ capturedErrors.length }} error{{ capturedErrors.length === 1 ? '' : 's' }} captured</span>
          <button class="studio-debug-indicator__clear" type="button" @click="clearErrors">
            Clear
          </button>
        </div>
        <div
          v-for="(err, i) in capturedErrors"
          :key="err.timestamp"
          class="studio-debug-indicator__error"
        >
          <button
            class="studio-debug-indicator__error-toggle"
            type="button"
            :aria-expanded="errorExpanded === i"
            @click="toggleErrorExpanded(i)"
          >
            <ChevronDown v-if="errorExpanded !== i" :size="11" :stroke-width="2.5" aria-hidden="true" />
            <ChevronUp v-else :size="11" :stroke-width="2.5" aria-hidden="true" />
            <span class="studio-debug-indicator__error-msg">{{ err.message }}</span>
          </button>
          <pre v-if="errorExpanded === i && err.stack" class="studio-debug-indicator__stack">{{ err.stack }}</pre>
        </div>
      </div>

      <!-- Info rows -->
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

      <!-- Footer -->
      <div class="studio-debug-indicator__footer">
        <button class="studio-debug-indicator__btn" type="button" @click="copyBundle">
          <Copy :size="12" :stroke-width="2" aria-hidden="true" />
          {{ copyLabel }}
        </button>
        <button
          class="studio-debug-indicator__btn studio-debug-indicator__btn--muted"
          type="button"
          @click="open = false"
        >
          Close
        </button>
      </div>
    </div>

    <!-- Trigger badge -->
    <button
      class="studio-debug-indicator__trigger"
      type="button"
      :aria-expanded="open"
      aria-label="Open Studio debug indicator"
      @click="toggleOpen"
    >
      <span
        v-if="hasErrors"
        class="studio-debug-indicator__error-dot"
        aria-label="Errors present"
      />
      <span class="studio-debug-indicator__mark" aria-hidden="true">d</span>
    </button>
  </div>
</template>

<style scoped>
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

/* Trigger */
.studio-debug-indicator__trigger {
  position: relative;
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(255 255 255 / 0.08);
  border-radius: 10px;
  background: #0a0a0a;
  box-shadow:
    0 0 0 1px rgb(0 0 0 / 0.08),
    0 8px 24px rgb(0 0 0 / 0.18);
  color: #fafafa;
  cursor: pointer;
  transition: transform 120ms ease, box-shadow 120ms ease;
}

.studio-debug-indicator__trigger:hover {
  transform: translateY(-1px);
  box-shadow:
    0 0 0 1px rgb(0 0 0 / 0.1),
    0 12px 28px rgb(0 0 0 / 0.22);
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
  background: oklch(0.55 0.15 28);
  box-shadow: 0 0 0 1.5px #0a0a0a;
}

/* Panel */
.studio-debug-indicator__panel {
  width: min(320px, calc(100vw - 32px));
  overflow: hidden;
  border: 1px solid rgb(255 255 255 / 0.08);
  border-radius: 12px;
  background: #0a0a0a;
  box-shadow:
    0 0 0 1px rgb(0 0 0 / 0.2),
    0 18px 40px rgb(0 0 0 / 0.28);
  color: #fafafa;
}

.studio-debug-indicator__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border-bottom: 1px solid rgb(255 255 255 / 0.08);
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
  color: #737373;
  cursor: pointer;
}

.studio-debug-indicator__icon-btn:hover {
  background: rgb(255 255 255 / 0.08);
  color: #fafafa;
}

/* Errors section */
.studio-debug-indicator__errors {
  border-bottom: 1px solid rgb(255 255 255 / 0.08);
}

.studio-debug-indicator__errors-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px 6px;
  color: oklch(0.72 0.15 28);
  font-size: 11px;
  font-weight: 600;
}

.studio-debug-indicator__clear {
  margin-left: auto;
  border: 0;
  background: transparent;
  color: #737373;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.studio-debug-indicator__clear:hover {
  color: #fafafa;
}

.studio-debug-indicator__error {
  border-top: 1px solid rgb(255 255 255 / 0.05);
}

.studio-debug-indicator__error-toggle {
  display: flex;
  width: 100%;
  align-items: flex-start;
  gap: 6px;
  border: 0;
  background: transparent;
  color: #fafafa;
  padding: 7px 12px;
  text-align: left;
  cursor: pointer;
}

.studio-debug-indicator__error-toggle:hover {
  background: rgb(255 255 255 / 0.04);
}

.studio-debug-indicator__error-toggle svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: #737373;
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
  border-top: 1px solid rgb(255 255 255 / 0.05);
  background: rgb(255 255 255 / 0.03);
  color: #a1a1a1;
  padding: 8px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

/* Info rows */
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
  color: #737373;
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
  color: #fafafa;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 550;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.studio-debug-indicator__path {
  overflow: hidden;
  color: #737373;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Footer */
.studio-debug-indicator__footer {
  display: flex;
  gap: 6px;
  border-top: 1px solid rgb(255 255 255 / 0.08);
  padding: 10px 12px;
}

.studio-debug-indicator__btn {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid rgb(255 255 255 / 0.1);
  border-radius: 7px;
  background: rgb(255 255 255 / 0.06);
  color: #fafafa;
  padding: 0 10px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.studio-debug-indicator__btn:hover {
  background: rgb(255 255 255 / 0.1);
}

.studio-debug-indicator__btn--muted {
  margin-left: auto;
  border-color: transparent;
  background: transparent;
  color: #737373;
}

.studio-debug-indicator__btn--muted:hover {
  background: rgb(255 255 255 / 0.06);
  color: #fafafa;
}

@media (prefers-reduced-motion: reduce) {
  .studio-debug-indicator__trigger {
    transition: none;
  }
}
</style>
