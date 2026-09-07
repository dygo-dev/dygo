<script setup lang="ts">
import { Button, PasswordField } from '@/design'

defineProps<{
  id: string
  label: string
  modelValue?: unknown
  present?: boolean
  required?: boolean
  disabled?: boolean
  error?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string | null | undefined] }>()
</script>

<template>
  <div class="grid gap-1">
    <PasswordField
      :id="id"
      :label="label"
      :model-value="typeof modelValue === 'string' ? modelValue : ''"
      :hint="modelValue === null ? 'Will be cleared' : present === undefined ? 'Status unavailable' : present ? 'Set' : 'Not set'"
      :disabled="disabled"
      :error="error"
      autocomplete="new-password"
      @update:model-value="emit('update:modelValue', $event === '' ? undefined : $event)"
    />
    <Button v-if="!required" type="button" variant="ghost" size="xs" :disabled="disabled" @click="emit('update:modelValue', null)">Clear</Button>
  </div>
</template>
