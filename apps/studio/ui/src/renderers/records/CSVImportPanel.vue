<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'

import { Button } from '@/design'
import type { MetadataField } from '@/features/metadata/metadata.api'
import { getImportStatus, startCSVImport } from '@/features/records/records.api'
import { parseCSV, recordsToCSV } from '@/features/records/csv'

const props = defineProps<{
  entity: string
  appName: string
  entityKey: string
  fields: MetadataField[]
}>()
const emit = defineEmits<{ close: []; imported: [] }>()

const fileInput = ref<HTMLInputElement | null>(null)
const headers = ref<string[]>([])
const sourceRows = ref<string[][]>([])
const mapping = ref<Record<string, string>>({})
const rowErrors = ref<Record<number, string[]>>({})
const progress = ref(0)
const importing = ref(false)
const error = ref('')
let pollTimer: ReturnType<typeof window.setTimeout> | undefined
const importableFields = computed(() => props.fields.filter((field) => field.stored && !field['write-only'] && field.type !== 'collection'))
const validRows = computed(() => sourceRows.value.slice(1).map((row, index) => ({ row, index: index + 2 })).filter(({ row }) => row.some((value) => value.trim() !== '')))

function openFilePicker() { fileInput.value?.click() }

function selectFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  error.value = ''
  rowErrors.value = {}
  const reader = new FileReader()
  reader.onload = () => {
    const parsed = parseCSV(String(reader.result ?? ''))
    headers.value = parsed[0] ?? []
    sourceRows.value = parsed
    mapping.value = Object.fromEntries(headers.value.map((header) => [header, matchingField(header)]))
    validateRows()
  }
  reader.onerror = () => { error.value = 'The CSV file could not be read.' }
  reader.readAsText(file)
}

function matchingField(header: string): string {
  const normalized = header.trim().toLowerCase()
  return importableFields.value.find((field) => field.name.toLowerCase() === normalized || field.label.toLowerCase() === normalized)?.name ?? ''
}

function validateRows() {
  const mapped = Object.values(mapping.value).filter(Boolean)
  if (new Set(mapped).size !== mapped.length) {
    error.value = 'Each field can be mapped only once.'
    return
  }
  error.value = ''
  const required = importableFields.value.filter((field) => field.required).map((field) => field.name)
  const errors: Record<number, string[]> = {}
  validRows.value.forEach(({ row, index }) => {
    const values = Object.fromEntries(headers.value.map((header, column) => [mapping.value[header], row[column]?.trim() ?? '']))
    const missing = required.filter((field) => !values[field])
    if (missing.length > 0) errors[index] = missing.map((field) => `${field} is required`)
  })
  rowErrors.value = errors
}

async function importRows() {
  validateRows()
  if (error.value || Object.keys(rowErrors.value).length > 0 || validRows.value.length === 0 || importing.value) return
  importing.value = true
  progress.value = 0
  error.value = ''
  const columns = headers.value.flatMap((header) => mapping.value[header] ? [{ source: header, key: mapping.value[header] }] : [])
  const rows = validRows.value.map(({ row }) => Object.fromEntries(columns.map((column) => [column.key, row[headers.value.indexOf(column.source)]?.trim() ?? ''])))
  const csv = recordsToCSV(rows, columns.map((column) => ({ key: column.key, label: column.key })))
  try {
    const queued = await startCSVImport(props.appName, props.entityKey, new Blob([csv], { type: 'text/csv' }))
    await pollImport(queued.id)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : 'Import could not start.'
    importing.value = false
  }
}

async function pollImport(id: number) {
  const result = await getImportStatus(id)
  progress.value = result['total-rows'] > 0 ? Math.round((result['processed-rows'] / result['total-rows']) * 100) : 100
  rowErrors.value = Object.fromEntries((result.rows ?? []).filter((row) => row.error).map((row) => [row['row-number'] + 1, [row.error ?? 'Record could not be created.']]))
  if (result.status === 'queued' || result.status === 'running') {
    await new Promise<void>((resolve) => { pollTimer = window.setTimeout(resolve, 1000) })
    return pollImport(id)
  }
  importing.value = false
  emit('imported')
  if (result.status === 'succeeded') emit('close')
}

onBeforeUnmount(() => { if (pollTimer) window.clearTimeout(pollTimer) })
</script>

<template>
  <section class="csv-import" aria-labelledby="csv-import-title">
    <div class="csv-import__heading"><div><h2 id="csv-import-title">Import CSV</h2><p>Map CSV columns to fields. Required values are checked before import.</p></div><Button variant="ghost" size="sm" @click="emit('close')">Close</Button></div>
    <input ref="fileInput" class="csv-import__file" type="file" accept=".csv,text/csv" @change="selectFile" />
    <Button variant="secondary" size="sm" :disabled="importing" @click="openFilePicker">Choose CSV file</Button>
    <div v-if="headers.length > 0" class="csv-import__mapping">
      <label v-for="header in headers" :key="header">{{ header }}
        <select v-model="mapping[header]" @change="validateRows">
          <option value="">Skip column</option><option v-for="field in importableFields" :key="field.name" :value="field.name">{{ field.label }}</option>
        </select>
      </label>
    </div>
    <p v-if="error" class="csv-import__error" role="alert">{{ error }}</p>
    <p v-if="validRows.length > 0" class="csv-import__summary">{{ validRows.length }} rows · {{ progress }}% complete</p>
    <ul v-if="Object.keys(rowErrors).length > 0" class="csv-import__errors"><li v-for="(messages, row) in rowErrors" :key="row">Row {{ row }}: {{ messages.join(', ') }}</li></ul>
    <Button variant="primary" size="sm" :disabled="Boolean(error) || validRows.length === 0 || Object.keys(rowErrors).length > 0" :loading="importing" @click="importRows">Import records</Button>
  </section>
</template>

<style scoped>
.csv-import { display: grid; gap: 12px; border: 1px solid var(--studio-border); background: var(--studio-surface); padding: 14px; }
.csv-import__heading { display: flex; justify-content: space-between; gap: 12px; }
.csv-import h2 { margin: 0; font-size: 15px; }
.csv-import p { margin: 4px 0 0; color: var(--studio-text-muted); font-size: 12px; }
.csv-import__file { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
.csv-import__mapping { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 8px; }
.csv-import__mapping label { display: grid; gap: 4px; color: var(--studio-text-muted); font-size: 11px; }
.csv-import select { min-height: var(--studio-control-height-sm); border: 1px solid var(--studio-border); border-radius: var(--studio-radius-control); background: var(--studio-control-bg); color: var(--studio-text); padding: 0 6px; }
.csv-import__errors, .csv-import__error { margin: 0; color: var(--studio-danger); font-size: 12px; }
.csv-import__errors { padding-left: 18px; }
</style>
