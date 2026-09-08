import { defineAsyncComponent } from 'vue'

export const recordViews = [
  { id: 'list', label: 'List', requiresTree: false, renderer: null },
  { id: 'tree', label: 'Tree', requiresTree: true, renderer: defineAsyncComponent(() => import('../../renderers/records/RecordTreeView.vue')) },
] as const

export type RecordViewID = typeof recordViews[number]['id']
