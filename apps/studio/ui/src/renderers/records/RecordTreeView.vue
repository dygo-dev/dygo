<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQueries } from '@tanstack/vue-query'
import { TreeRoot, TreeItem, TreeVirtualizer } from 'reka-ui'
import { ChevronRight } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth.store'
import { listTreeRecords } from '@/features/records/tree.api'
import { searchTree, treeRecord, type TreeItem as Item } from '@/features/records/tree'
import type { RecordData } from '@/features/records/records.api'
import type { RecordListRouteState } from '@/features/records/query'

const props = defineProps<{ entity: string; tree: { 'parent-field': string; 'label-field'?: string }; query: RecordListRouteState; pageSize: number }>()
const emit = defineEmits<{ 'open-record': [record: RecordData] }>()
const auth = useAuthStore()
const expanded = ref<string[]>([])
const pages = ref<Record<string, number>>({})
const filtered = computed(() => props.query.filters.length > 0)
watch(() => [props.entity, props.query, props.pageSize, auth.sessionVersion], () => { expanded.value = []; pages.value = {} }, { deep: true })
const requests = computed(() => ['', ...(!filtered.value ? expanded.value.map((key) => key.slice('record:'.length)) : [])].flatMap((parent) => Array.from({ length: (pages.value[parent] ?? 0) + 1 }, (_, page) => ({ parent, page }))))
const queries = useQueries({ queries: computed(() => requests.value.map(({ parent, page }) => ({
  queryKey: ['records', 'tree', auth.currentUser?.id, auth.sessionVersion, props.entity, props.query, props.pageSize, parent, page],
  queryFn: ({ signal }: { signal: AbortSignal }) => listTreeRecords(props.entity, filtered.value ? 'search' : parent ? 'children' : 'roots', { ...props.query, limit: props.pageSize, offset: page * props.pageSize }, { name: parent, signal }),
  enabled: !!auth.currentUser && props.pageSize > 0,
}))) })

function branch(parent: string): Item[] {
  const rows: Item[] = []
  requests.value.forEach((request, index) => {
    if (request.parent !== parent) return
    const query = queries.value[index]
    if (!query) { rows.push({ key: `loading:${parent}`, label: 'Loading…' }); return }
    for (const node of query.data?.data ?? []) rows.push(treeRecord({ ...node, matched: true }, props.tree['label-field']))
    if (query.isPending) rows.push({ key: `loading:${parent}`, label: 'Loading…' })
    else if (query.error) rows.push({ key: `error:${parent}`, label: `${query.error.message} — Retry`, action: () => { void query.refetch() } })
    else if (request.page === (pages.value[parent] ?? 0) && query.data && (query.data.meta.total !== undefined ? query.data.meta.offset + query.data.meta.count < query.data.meta.total : query.data.data.length === props.pageSize)) {
      rows.push({ key: `more:${parent}`, label: 'Load more', action: () => { pages.value[parent] = request.page + 1 } })
    }
  })
  return rows
}
const items = computed(() => {
  if (filtered.value) {
    const matches = queries.value.flatMap((query) => query.data?.data ?? [])
    const context = queries.value.flatMap((query) => query.data?.context ?? [])
    return [...searchTree(matches, context, props.tree['parent-field'], props.tree['label-field']), ...branch('').filter((item) => !item.record)]
  }
  const roots = branch('')
  const nodes = new Map<string, Item>(roots.filter((item) => item.record).map((item) => [item.key, item]))
  // Expansion order need not be parent-first after keyboard navigation.
  const children = new Map(expanded.value.map((key) => [key, branch(key.slice('record:'.length))]))
  for (const rows of children.values()) for (const row of rows) if (row.record) nodes.set(row.key, row)
  for (const [parent, rows] of children) { const node = nodes.get(parent); if (node) node.children = rows }
  return roots
})
watch(items, () => {
  if (!filtered.value) return
  const keys: string[] = []
  const visit = (nodes: Item[]) => nodes.forEach((node) => { if (node.children?.length) { keys.push(node.key); visit(node.children) } })
  visit(items.value)
  expanded.value = [...new Set([...expanded.value, ...keys])]
})
function activate(item: Item) {
  if (item.action) item.action()
  else if (item.record) emit('open-record', item.record)
}
function toggle(key: string) { expanded.value = expanded.value.includes(key) ? expanded.value.filter((item) => item !== key) : [...expanded.value, key] }
</script>

<template>
  <div class="record-tree-view">
    <p v-if="!items.length" role="status">{{ filtered ? 'No matching Records' : 'No Records' }}</p>
    <TreeRoot v-else v-model:expanded="expanded" :items="items" :get-key="(item: Item) => item.key" :get-children="(item: Item) => item.children" aria-label="Record tree" class="record-tree-view__tree">
      <TreeVirtualizer v-slot="{ item }" :estimate-size="34" :text-content="(item) => String(item.label)">
        <TreeItem v-slot="{ isExpanded }" v-bind="item.bind" :value="item.value" class="record-tree-view__row" :style="{ paddingLeft: `${12 + (item.level - 1) * 20}px` }" @select.prevent="activate(item.value as Item)" @toggle="(event) => { if (event.detail.originalEvent.type === 'click') event.preventDefault() }">
          <button v-if="item.value.children" type="button" tabindex="-1" :aria-label="`${isExpanded ? 'Collapse' : 'Expand'} ${item.value.label}`" @click.stop="toggle(item.value.key)"><ChevronRight :size="14" :class="{ expanded: isExpanded }" aria-hidden="true" /></button>
          <span v-else class="record-tree-view__spacer" />
          <span class="record-tree-view__label">{{ item.value.label }}</span>
          <small v-if="item.value.pathUnavailable">Path unavailable</small>
          <small v-else-if="item.value.contextOnly">Context</small>
        </TreeItem>
      </TreeVirtualizer>
    </TreeRoot>
  </div>
</template>

<style scoped>
.record-tree-view { flex: 1; min-height: 0; overflow: hidden; }
.record-tree-view > p { padding: 20px; color: var(--studio-text-muted); }
.record-tree-view__tree { height: 100%; overflow: auto; padding: 4px 0; margin: 0; list-style: none; }
.record-tree-view__row { display: flex; align-items: center; gap: 8px; height: 34px; padding-right: 12px; color: var(--studio-text); cursor: pointer; }
.record-tree-view__row:hover { background: var(--studio-surface-raised); }
.record-tree-view__row:focus-visible { outline: 2px solid var(--studio-focus); outline-offset: -2px; background: var(--studio-surface-raised); }
.record-tree-view__row button { display: grid; place-items: center; flex: none; width: 20px; height: 26px; border: 0; background: transparent; color: var(--studio-text-muted); cursor: pointer; }
.record-tree-view__row .expanded { transform: rotate(90deg); }
.record-tree-view__spacer { width: 20px; flex: none; }
.record-tree-view__label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.record-tree-view__row small { color: var(--studio-text-muted); white-space: nowrap; }
</style>
