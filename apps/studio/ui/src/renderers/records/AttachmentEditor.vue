<script setup lang="ts">
import { ref } from 'vue'

import { Button } from '@/design'

const props = withDefaults(defineProps<{
  id: string
  label: string
  modelValue: string
  disabled?: boolean
  readonly?: boolean
  error?: string
  upload?: (file: File) => Promise<string>
}>(), {
  disabled: false,
  readonly: false,
  error: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'upload-request': [file: File]
}>()

const input = ref<HTMLInputElement | null>(null)
const selecting = ref(false)
const uploadError = ref('')

function selectFile() {
  input.value?.click()
}

async function fileSelected(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  selecting.value = true
  uploadError.value = ''
  try {
    if (props.upload) {
      emit('update:modelValue', await props.upload(file))
    } else {
      emit('upload-request', file)
    }
  } catch (error) {
    uploadError.value = error instanceof Error ? error.message : 'File upload failed.'
  } finally {
    selecting.value = false
  }
}

function clearValue() {
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="attachment-editor">
    <label class="attachment-editor__label" :for="id">{{ label }}</label>
    <div class="attachment-editor__row">
      <input
        :id="id"
        ref="input"
        class="attachment-editor__file"
        type="file"
        :disabled="disabled || readonly"
        @change="fileSelected"
      />
      <Button type="button" variant="secondary" size="sm" :disabled="disabled || readonly || !upload" :loading="selecting" @click="selectFile">
        Choose file
      </Button>
      <Button v-if="modelValue" type="button" variant="ghost" size="sm" :disabled="disabled || readonly" @click="clearValue">
        Clear
      </Button>
    </div>
    <p v-if="modelValue" class="attachment-editor__value"><a :href="`/api/v1/files/${encodeURIComponent(modelValue)}`">Download file</a></p>
    <p class="attachment-editor__hint">{{ upload ? 'Upload a private file to this Record.' : 'Save this Record before you upload a file.' }}</p>
    <p v-if="error || uploadError" class="attachment-editor__error" role="alert">{{ error || uploadError }}</p>
  </div>
</template>

<style scoped>
.attachment-editor { display: grid; gap: 6px; }
.attachment-editor__label { color: var(--studio-text); font-size: 13px; font-weight: 600; }
.attachment-editor__row { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
.attachment-editor__file { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
.attachment-editor__value { margin: 0; color: var(--studio-text); font-size: 13px; overflow-wrap: anywhere; }
.attachment-editor__hint { margin: 0; color: var(--studio-text-muted); font-size: 12px; }
.attachment-editor__error { margin: 0; color: var(--studio-danger); font-size: 12px; }
</style>
