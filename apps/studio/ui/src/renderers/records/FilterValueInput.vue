<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { ComboboxRoot, ComboboxInput, ComboboxContent, ComboboxItem, ComboboxEmpty, ComboboxPortal } from 'reka-ui'
import { linkOptions, type MetadataField } from '@/features/metadata/metadata.api'
import { useMetadataEntitiesQuery } from '@/features/metadata/metadata.query'
import { listRecords, type RecordData } from '@/features/records/records.api'
import { filterDisplayValue, filterInputType, filterWireValue } from '@/features/records/filter-values'
import { resolveLinkFilters } from './link-options'
import { selectOptions } from './record-field-utils'
import { useAuthStore } from '@/stores/auth.store'

const props = defineProps<{ field: MetadataField; operator: string; modelValue: string; currentValues: RecordData; appName: string }>()
const auth = useAuthStore()
const emit = defineEmits<{ 'update:modelValue': [value: string]; apply: [] }>()
const arity = computed(() => props.field.filter.operators.find((item) => item.key === props.operator)?.arity)
const values = computed(() => arity.value === 'range' ? props.modelValue.split('..') : [props.modelValue])
const link = computed(() => linkOptions(props.field))
const entities = useMetadataEntitiesQuery()
const target = computed(() => {
  const matches = (entities.data.value ?? []).filter((entity) => entity.key === link.value?.entity && entity.app.name === (link.value?.app || props.appName))
  return matches.length === 1 ? matches[0]?.slug : undefined
})
const constraints = computed(() => resolveLinkFilters(link.value?.filters ?? [], props.currentValues))
const canSearch = computed(() => !!target.value && constraints.value.length === (link.value?.filters.length ?? 0))
const search = ref('')
const open = ref(false)
const records = useQuery({
  queryKey: computed(() => ['records', 'filter-options', auth.currentUser?.id, target.value, constraints.value, search.value]),
  queryFn: ({ signal }) => listRecords(target.value!, { limit: 20, offset: 0, filters: [...constraints.value, ...(search.value ? [{ field: 'name', operator: 'contains', value: search.value }] : [])] }, { signal }),
  enabled: computed(() => !!auth.currentUser && props.field.type === 'link' && open.value && canSearch.value),
})
function update(value: string, index: number) {
  const next = [values.value[0] ?? '', values.value[1] ?? '']
  next[index] = filterWireValue(value, props.field.type)
  emit('update:modelValue', arity.value === 'range' ? next.join('..') : next[0]!)
}
</script>

<template>
  <span v-if="arity !== 'none'" class="filter-value-input">
    <ComboboxRoot v-if="field.type === 'link'" v-model:open="open" :model-value="modelValue" :disabled="!canSearch" ignore-filter @update:model-value="(value) => { emit('update:modelValue', String(value)); emit('apply') }">
      <ComboboxInput :display-value="() => modelValue" :aria-label="`${field.label} value`" :placeholder="canSearch ? 'Search records' : 'Set dependent filters first'" @input="search = ($event.target as HTMLInputElement).value" />
      <ComboboxPortal><ComboboxContent class="filter-field-picker" position="popper">
        <span v-if="records.isFetching.value">Loading…</span>
        <span v-else-if="records.error.value">Could not load records.</span>
        <ComboboxEmpty v-else>No matching records</ComboboxEmpty>
        <ComboboxItem v-for="record in records.data.value?.data ?? []" :key="String(record.name)" :value="String(record.name)">{{ link?.displayField ? record[link.displayField] ?? record.name : record.name }}</ComboboxItem>
      </ComboboxContent></ComboboxPortal>
    </ComboboxRoot>
    <template v-else v-for="index in arity === 'range' ? 2 : 1" :key="index">
      <span v-if="index === 2">–</span>
      <select v-if="field.type === 'boolean' || field.type === 'select'" :value="values[index - 1] ?? ''" :aria-label="`${field.label} value ${index}`" @change="update(($event.target as HTMLSelectElement).value, index - 1); emit('apply')">
        <option value="">Choose</option>
        <option v-for="option in field.type === 'boolean' ? [{value:'true',label:'True'}, {value:'false',label:'False'}] : selectOptions(field)" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <input v-else :value="filterDisplayValue(values[index - 1] ?? '', field.type)" :type="filterInputType(field)" :step="['int', 'bigint'].includes(field.type) ? '1' : 'any'" :aria-label="`${field.label} ${arity === 'range' ? index === 1 ? 'start' : 'end' : 'value'}`" @input="update(($event.target as HTMLInputElement).value, index - 1)" @keydown.enter.prevent="emit('apply')" />
    </template>
  </span>
</template>

<style scoped>
.filter-value-input { display: inline-flex; align-items: center; min-width: 0; }
.filter-value-input :is(input, select) { width: 140px; min-width: 70px; height: 100%; border: 0; background: var(--studio-surface); color: var(--studio-text); padding: 0 6px; font: inherit; }
</style>
