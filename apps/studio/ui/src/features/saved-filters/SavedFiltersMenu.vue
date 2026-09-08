<script setup lang="ts">
import { ref } from 'vue'
import { PopoverRoot, PopoverTrigger, PopoverPortal, PopoverContent } from 'reka-ui'
import { Button, Input } from '@/design'
import type { SavedFilter } from './saved-filters.api'
defineProps<{ items: SavedFilter[]; busy: boolean; error: string }>()
const emit = defineEmits<{ apply: [item: SavedFilter]; create: [label: string]; rename: [item: SavedFilter, label: string]; replace: [item: SavedFilter]; delete: [item: SavedFilter]; retry: [] }>()
const label = ref('')
const editing = ref<number | null>(null)
const newLabel = ref('')
</script>
<template>
  <PopoverRoot><PopoverTrigger as-child><Button variant="ghost" size="sm">Saved filters</Button></PopoverTrigger>
    <PopoverPortal><PopoverContent class="saved-filters-menu" align="start" :side-offset="6">
      <p v-if="error" role="alert">{{ error }} <Button variant="ghost" size="sm" @click="emit('retry')">Retry</Button></p>
      <p v-else-if="busy" role="status">Loading…</p>
      <p v-else-if="!items.length">No saved filters</p>
      <div v-for="item in items" :key="item.id" class="saved-filters-menu__item">
        <form v-if="editing === item.id" @submit.prevent="emit('rename', item, newLabel.trim()); editing = null">
          <Input v-model="newLabel" aria-label="Filter name" required /><Button type="submit" size="sm" :disabled="busy || !newLabel.trim()">Rename</Button>
        </form>
        <Button v-else variant="ghost" size="sm" :disabled="busy || !!item.validationError" @click="emit('apply', item)">{{ item.label }}</Button>
        <p v-if="item.validationError" role="status">{{ item.validationError }}</p>
        <div>
          <Button variant="ghost" size="sm" :disabled="busy" @click="editing = item.id; newLabel = item.label">Rename</Button>
          <Button variant="ghost" size="sm" :disabled="busy" @click="emit('replace', item)">Replace</Button>
          <Button variant="ghost" size="sm" :disabled="busy" @click="emit('delete', item)">Delete</Button>
        </div>
      </div>
      <form @submit.prevent="emit('create', label.trim())">
        <Input v-model="label" placeholder="Filter name" aria-label="New saved filter name" required />
        <Button type="submit" size="sm" :disabled="busy || !label.trim()">Save current filters</Button>
      </form>
    </PopoverContent></PopoverPortal>
  </PopoverRoot>
</template>
<style>
.saved-filters-menu { z-index: 60; width: 340px; max-height: 420px; overflow: auto; padding: 12px; border: 1px solid var(--studio-border); border-radius: var(--studio-radius-control); background: var(--studio-surface); color: var(--studio-text); box-shadow: var(--studio-shadow-sheet); }
.saved-filters-menu form { display: flex; gap: 6px; }
.saved-filters-menu__item { border-bottom: 1px solid var(--studio-border); margin-bottom: 8px; padding-bottom: 8px; }
</style>
