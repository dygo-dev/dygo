<script setup lang="ts">
import { computed, ref } from 'vue'
import { ComboboxRoot, ComboboxInput, ComboboxContent, ComboboxItem, ComboboxEmpty, PopoverRoot, PopoverTrigger, PopoverPortal, PopoverContent } from 'reka-ui'
import { FunnelPlus } from '@lucide/vue'
import { IconButton } from '@/design'
import type { MetadataField } from '@/features/metadata/metadata.api'
const props = defineProps<{ fields: MetadataField[] }>()
const emit = defineEmits<{ select: [field: string] }>()
const open = ref(false)
const search = ref('')
const matches = computed(() => props.fields.filter((field) => `${field.label} ${field.name}`.toLowerCase().includes(search.value.toLowerCase())))
</script>

<template>
  <PopoverRoot v-model:open="open" @update:open="search = ''">
    <PopoverTrigger as-child><IconButton label="Add filter"><FunnelPlus :size="14" /></IconButton></PopoverTrigger>
    <PopoverPortal><PopoverContent class="filter-field-picker" :side-offset="6" align="start">
      <ComboboxRoot :open="true" ignore-filter @update:model-value="(value) => { emit('select', String(value)); open = false }">
        <ComboboxInput v-model="search" placeholder="Search fields" aria-label="Search filter fields" />
        <ComboboxContent><ComboboxEmpty>No matching fields</ComboboxEmpty>
          <ComboboxItem v-for="field in matches" :key="field.name" :value="field.name">{{ field.label || field.name }}</ComboboxItem>
        </ComboboxContent>
      </ComboboxRoot>
    </PopoverContent></PopoverPortal>
  </PopoverRoot>
</template>

<style>
.filter-field-picker { z-index: 60; width: 240px; max-height: 320px; overflow: auto; padding: 8px; border: 1px solid var(--studio-border); border-radius: var(--studio-radius-control); background: var(--studio-surface); box-shadow: var(--studio-shadow-sheet); color: var(--studio-text); }
.filter-field-picker input { width: 100%; padding: 6px; background: var(--studio-control-bg); border: 1px solid var(--studio-border); color: inherit; }
.filter-field-picker [role=option] { padding: 6px; cursor: pointer; }
.filter-field-picker [data-highlighted] { background: var(--studio-surface-raised); }
</style>
