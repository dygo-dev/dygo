<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ExternalLink, LoaderCircle, Plus } from '@lucide/vue'
import { useQuery } from '@tanstack/vue-query'

import { Field, IconButton, Input } from '@/design'
import { useMetadataEntitiesQuery } from '@/features/metadata/metadata.query'
import { linkOptions, type MetadataField } from '@/features/metadata/metadata.api'
import { getRecordByName, listRecords, type RecordData } from '@/features/records/records.api'
import { resolveLinkFilters } from './link-options'

const props = withDefaults(defineProps<{
  id: string
  label: string
  field: MetadataField
  modelValue?: string
  currentValues?: RecordData
  hint?: string
  error?: string
  required?: boolean
  disabled?: boolean
  readonly?: boolean
}>(), {
  modelValue: '',
  currentValues: () => ({}),
  hint: '',
  error: '',
  required: false,
  disabled: false,
  readonly: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'open-related': [recordName: string]
  'create-related': []
}>()

const search = ref('')
const open = ref(false)
const activeIndex = ref(0)
const options = computed(() => linkOptions(props.field))
const entitiesQuery = useMetadataEntitiesQuery()
const target = computed(() => {
  const link = options.value
  if (!link) {
    return null
  }
  return (entitiesQuery.data.value ?? []).find((entity) => (
    entity.key === link.entity && (!link.app || entity.app.name === link.app)
  )) ?? null
})
const targetSlug = computed(() => target.value?.slug || options.value?.entity || '')
const targetFilters = computed(() => resolveLinkFilters(options.value?.filters ?? [], props.currentValues))
const listQuery = useQuery({
  queryKey: computed(() => ['records', 'link-options', targetSlug.value, search.value.trim(), targetFilters.value]),
  queryFn: ({ signal }) => listRecords(targetSlug.value, {
    limit: 20,
    offset: 0,
    filters: [
      ...targetFilters.value,
      ...(search.value.trim() ? [{ field: 'name', operator: 'contains', value: search.value.trim() }] : []),
    ],
  }, { signal }),
  enabled: computed(() => open.value && targetSlug.value !== ''),
})
const selectedQuery = useQuery({
  queryKey: computed(() => ['records', 'link-option', targetSlug.value, props.modelValue]),
  queryFn: ({ signal }) => getRecordByName(targetSlug.value, props.modelValue, { signal }),
  enabled: computed(() => targetSlug.value !== '' && props.modelValue.trim() !== ''),
})
const choices = computed(() => (listQuery.data.value?.data ?? []).flatMap((record) => {
  const name = recordName(record)
  if (!name) {
    return []
  }
  return [{ name, label: displayValue(record, options.value?.displayField) }]
}))
const selectedLabel = computed(() => {
  if (!props.modelValue) {
    return ''
  }
  const record = selectedQuery.data.value
  return record ? displayValue(record, options.value?.displayField) : props.modelValue
})
const listboxID = computed(() => `${props.id}-options`)
let filterCheck = 0

watch(targetFilters, async (filters) => {
  const version = ++filterCheck
  if (!props.modelValue || !targetSlug.value) return
  const result = await listRecords(targetSlug.value, { limit: 1, offset: 0, filters: [...filters, { field: 'name', operator: 'eq', value: props.modelValue }] }).catch(() => null)
  if (version === filterCheck && result && result.data.length === 0) clearValue()
}, { deep: true })

function updateSearch(value: string) {
  search.value = value
  open.value = true
  activeIndex.value = 0
}

function selectChoice(choice: { name: string; label: string }) {
  emit('update:modelValue', choice.name)
  search.value = choice.label
  open.value = false
}

function clearValue() {
  emit('update:modelValue', '')
  search.value = ''
  open.value = false
}

function closePicker() {
  window.setTimeout(() => {
    open.value = false
    search.value = ''
  }, 120)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    open.value = false
    search.value = ''
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    open.value = true
    activeIndex.value = Math.min(activeIndex.value + 1, Math.max(choices.value.length - 1, 0))
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
    return
  }
  if (event.key === 'Enter' && open.value && choices.value[activeIndex.value]) {
    event.preventDefault()
    selectChoice(choices.value[activeIndex.value])
  }
}

async function focusPicker(event: Event) {
  if (event.target instanceof HTMLInputElement && event.target.value === selectedLabel.value) {
    search.value = ''
  }
  open.value = true
  await nextTick()
  if (event.target instanceof HTMLInputElement && event.target.value === selectedLabel.value) {
    event.target.select()
  }
}

function recordName(record: RecordData): string {
  return typeof record.name === 'string' ? record.name : ''
}

function displayValue(record: RecordData, displayField?: string): string {
  const value = displayField ? record[displayField] : undefined
  if (typeof value === 'string' || typeof value === 'number') {
    return String(value)
  }
  return recordName(record)
}

</script>

<template>
  <Field
    :id="id"
    :label="label"
    :hint="hint"
    :error="error"
    :required="required"
    :disabled="disabled"
    :readonly="readonly"
  >
    <template #default="{ id: fieldId, invalid, describedBy }">
      <div class="link-picker">
        <div class="link-picker__input-row">
          <Input
            :id="fieldId"
            :model-value="open ? search : selectedLabel"
            :name="field.name"
            :placeholder="`Search ${label}`"
            :described-by="describedBy"
            :required="required"
            :disabled="disabled"
            :readonly="readonly"
            :invalid="invalid"
            role="combobox"
            aria-autocomplete="list"
            :aria-expanded="open"
            :aria-controls="listboxID"
            :aria-activedescendant="open && choices[activeIndex] ? `${listboxID}-${activeIndex}` : undefined"
            @focus="focusPicker"
            @click="focusPicker"
            @blur="closePicker"
            @keydown="onKeydown"
            @update:model-value="updateSearch"
          />
          <IconButton
            v-if="modelValue"
            label="Open related record"
            variant="ghost"
            :disabled="disabled"
            @click="emit('open-related', modelValue)"
          >
            <ExternalLink :size="14" stroke-width="1.8" aria-hidden="true" />
          </IconButton>
          <IconButton
            v-if="!readonly"
            label="Create related record"
            variant="ghost"
            :disabled="disabled"
            @click="emit('create-related')"
          >
            <Plus :size="14" stroke-width="1.8" aria-hidden="true" />
          </IconButton>
        </div>
        <div v-if="open" :id="listboxID" class="link-picker__menu" role="listbox" :aria-label="`${label} options`">
          <div v-if="listQuery.isPending.value" class="link-picker__state">
            <LoaderCircle class="link-picker__spinner" :size="14" aria-hidden="true" /> Loading options
          </div>
          <div v-else-if="listQuery.error.value" class="link-picker__state">Could not load options.</div>
          <button
            v-for="(choice, index) in choices"
            :id="`${listboxID}-${index}`"
            :key="choice.name"
            class="link-picker__option"
            :class="{ 'link-picker__option--active': index === activeIndex }"
            type="button"
            role="option"
            :aria-selected="choice.name === modelValue"
            @mousedown.prevent="selectChoice(choice)"
            @mouseenter="activeIndex = index"
          >
            <span>{{ choice.label }}</span>
            <small v-if="choice.label !== choice.name">{{ choice.name }}</small>
          </button>
          <div v-if="!listQuery.isPending.value && choices.length === 0" class="link-picker__state">No matching records.</div>
          <button v-if="modelValue" class="link-picker__clear" type="button" @mousedown.prevent="clearValue">Clear selection</button>
        </div>
      </div>
    </template>
  </Field>
</template>

<style scoped>
.link-picker { position: relative; }
.link-picker__input-row { display: flex; align-items: center; gap: 4px; }
.link-picker__input-row :deep(.d-input) { min-width: 0; }
.link-picker__menu {
  position: absolute;
  z-index: 30;
  top: calc(100% + 5px);
  right: 0;
  left: 0;
  display: grid;
  max-height: 260px;
  overflow: auto;
  border: 1px solid var(--studio-border);
  border-radius: var(--studio-radius-control);
  background: var(--studio-surface);
  box-shadow: var(--studio-shadow);
  padding: 4px;
}
.link-picker__option, .link-picker__clear {
  display: flex;
  width: 100%;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  border: 0;
  border-radius: var(--studio-radius-control);
  background: transparent;
  color: var(--studio-text);
  padding: 8px;
  text-align: left;
  font: inherit;
}
.link-picker__option:hover, .link-picker__option--active { background: var(--studio-surface-raised); }
.link-picker__option small { color: var(--studio-text-subtle); }
.link-picker__clear { border-top: 1px solid var(--studio-border); color: var(--studio-text-muted); font-size: 12px; }
.link-picker__state { display: flex; align-items: center; gap: 6px; color: var(--studio-text-muted); padding: 10px 8px; font-size: 12px; }
.link-picker__spinner { animation: link-picker-spin 700ms linear infinite; }
@keyframes link-picker-spin { to { transform: rotate(360deg); } }
</style>
