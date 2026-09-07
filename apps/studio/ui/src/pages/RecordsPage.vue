<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Plus } from '@lucide/vue'

import { ErrorState, Spinner } from '@/design'
import { useMetadataEntityMetaQuery } from '@/features/metadata/metadata.query'
import PageHeader from '@/shell/PageHeader.vue'
import type { PageHeaderAction } from '@/shell/types'
import { RecordListRenderer } from '@/renderers/records'
import { RouteName } from '@/router/routes'
import { humanizeEntity } from '@/stores/metadata.identity'
import { statusForError, storeError, type LoadStatus } from '@/stores/status'
import type { PinnedItem } from '@/features/pinned/pinned'
import RecordFormPage from './RecordFormPage.vue'
import { recordViews } from '@/features/records/views'
import { usePreferencesStore } from '@/features/preferences/preferences.store'

const props = defineProps<{
  entity: string
}>()

const router = useRouter()
const preferences = usePreferencesStore()
const entityMetaQuery = useMetadataEntityMetaQuery(() => props.entity)

const entityMeta = computed(() => entityMetaQuery.data.value ?? null)
const views = computed(() => recordViews.filter((view) => !view.requiresTree || !!entityMeta.value?.tree))
const viewKey = computed(() => `studio.records.${entityMeta.value?.app.name}.${entityMeta.value?.key}.view`)
const selectedView = computed(() => views.value.find((view) => view.id === preferences.get(viewKey.value, 'list')) ?? recordViews[0])
function selectView(value: string) {
  if (views.value.some((view) => view.id === value)) preferences.set(viewKey.value, value)
}
const entityMetaError = computed(() => (
  entityMetaQuery.error.value
    ? storeError(entityMetaQuery.error.value, 'Studio could not load entity metadata.')
    : null
))
const entityMetaStatus = computed<LoadStatus>(() => {
  if (entityMetaQuery.isPending.value) {
    return 'loading'
  }

  if (entityMetaError.value) {
    return statusForError(entityMetaError.value)
  }

  return entityMeta.value ? 'ready' : 'idle'
})
const isSingle = computed(() => entityMeta.value?.['is-single'] === true)
const isSystem = computed(() => entityMeta.value?.['is-system'] === true)
const canShowList = computed(() => entityMetaStatus.value === 'ready' && !isSingle.value)

const entityLabel = computed(() => {
  return entityMeta.value?.label || humanizeEntity(props.entity)
})
const pinTarget = computed<PinnedItem | null>(() => entityMeta.value ? ({
  type: 'entity', app: entityMeta.value.app.name, entity: entityMeta.value.key,
  label: entityLabel.value, path: `/${props.entity}`,
}) : null)

function openNewRecord() {
  if (isSystem.value) {
    return
  }
  void router.push({ name: RouteName.RecordNew, params: { entity: props.entity } })
}

function openRecord(row: Record<string, unknown>) {
  const recordName = row.name
  if (typeof recordName !== 'string' || recordName.length === 0) {
    return
  }

  void router.push({ name: RouteName.RecordDetail, params: { entity: props.entity, recordName } })
}

const actions = computed<PageHeaderAction[]>(() => {
  const next: PageHeaderAction[] = []
  if (!isSystem.value) {
    next.push({
      label: 'New record',
      icon: Plus,
      variant: 'primary',
      disabled: entityMetaStatus.value !== 'ready',
      onSelect: openNewRecord,
    })
  }

  return next
})

</script>

<template>
  <RecordFormPage
    v-if="isSingle"
    :entity="props.entity"
    mode="single"
  />

  <section v-else class="studio-page records-page" :aria-label="entityLabel">
    <PageHeader
      :show-title="false"
      :system="isSystem"
      :actions="canShowList ? actions : []"
      :pin-target="pinTarget"
    />

    <div v-if="entityMetaStatus === 'loading' || entityMetaStatus === 'idle'" class="studio-page-state">
      <Spinner size="sm" label="Loading entity" />
      <p>Loading entity</p>
    </div>

    <ErrorState
      v-else-if="entityMetaError"
      title="Entity unavailable"
      :message="entityMetaError.message"
    />

    <RecordListRenderer
      v-else-if="canShowList"
      :entity="props.entity"
      :entity-label="entityLabel"
      :fields="entityMeta?.fields ?? []"
      :system-fields="entityMeta?.['system-fields'] ?? []"
      :actions="entityMeta?.actions ?? []"
      :app-name="entityMeta?.app.name ?? ''"
      :entity-key="entityMeta?.key ?? ''"
      :read-only="isSystem"
      :view-key="selectedView.id"
      @create-record="openNewRecord"
      @open-record="openRecord"
    >
      <template #view-selector>
        <select class="records-page__view" aria-label="Record view" :value="selectedView.id" @change="selectView(($event.target as HTMLSelectElement).value)">
          <option v-for="view in views" :key="view.id" :value="view.id">{{ view.label }}</option>
        </select>
      </template>
      <template v-if="selectedView.renderer" #records="{ query, pageSize, revision }">
        <component :is="selectedView.renderer" :key="`${entity}:${revision}`" :entity="entity" :tree="entityMeta!.tree!" :query="query" :page-size="pageSize" @open-record="openRecord" />
      </template>
    </RecordListRenderer>
  </section>
</template>

<style scoped>
.records-page__view {
  height: var(--studio-control-height-sm);
  border: 1px solid var(--studio-border);
  border-radius: var(--studio-radius-control);
  background: var(--studio-surface);
  color: var(--studio-text);
  padding: 0 8px;
  font: inherit;
}

.records-page {
  gap: 0;
  grid-template-rows: auto minmax(0, 1fr);
  height: 100%;
  min-height: 0;
}

</style>
