<script setup lang="ts">
import { computed } from 'vue'
import { Activity, Send } from '@lucide/vue'

import { Button, Spinner, TextareaField } from '@/design'
import type { ActivityEntry } from '@/features/records/records.api'

const props = withDefaults(defineProps<{
  entries?: ActivityEntry[]
  loading?: boolean
  error?: string
  comment?: string
  submitting?: boolean
}>(), {
  entries: () => [],
  loading: false,
  error: '',
  comment: '',
  submitting: false,
})

const emit = defineEmits<{
  'update:comment': [value: string]
  comment: []
}>()

const canComment = computed(() => props.comment.trim().length > 0 && !props.submitting)

function actorLabel(entry: ActivityEntry): string {
  return entry.actor?.['full-name'] || entry.actor?.email || 'System'
}

function entryLabel(entry: ActivityEntry): string {
  return entry.title || entry.operation || entry.kind || 'Activity'
}

function entryDate(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
  <section class="record-timeline" aria-labelledby="record-timeline-title">
    <div class="record-timeline__heading">
      <div>
        <p id="record-timeline-title" class="record-timeline__title">Timeline</p>
        <p class="record-timeline__hint">Record changes, approvals, comments, and notifications.</p>
      </div>
      <Activity :size="16" :stroke-width="1.8" aria-hidden="true" />
    </div>

    <form class="record-timeline__comment" @submit.prevent="canComment && emit('comment')">
      <TextareaField
        id="record-timeline-comment"
        label="Add comment"
        :model-value="comment"
        :disabled="submitting"
        :rows="3"
        placeholder="Write a comment"
        @update:model-value="emit('update:comment', $event)"
      />
      <Button type="submit" variant="secondary" size="sm" :disabled="!canComment" :loading="submitting">
        <Send :size="14" :stroke-width="1.8" aria-hidden="true" />
        Add comment
      </Button>
    </form>

    <p v-if="error" class="record-timeline__error" role="alert">{{ error }}</p>
    <div v-else-if="loading" class="record-timeline__state" aria-live="polite">
      <Spinner size="sm" />
      Loading timeline
    </div>
    <p v-else-if="entries.length === 0" class="record-timeline__state">No activity yet.</p>
    <ol v-else class="record-timeline__entries">
      <li v-for="entry in entries" :key="entry.id" class="record-timeline__entry">
        <div class="record-timeline__marker" aria-hidden="true" />
        <div class="record-timeline__entry-body">
          <div class="record-timeline__entry-meta">
            <strong>{{ entryLabel(entry) }}</strong>
            <span>{{ actorLabel(entry) }}</span>
            <time :datetime="entry['created-at']" :title="entry['created-at']">{{ entryDate(entry['created-at']) }}</time>
          </div>
          <p v-if="entry.message" class="record-timeline__message">{{ entry.message }}</p>
          <div class="record-timeline__tags">
            <span>{{ entry.kind }}</span>
            <span>{{ entry.operation }}</span>
          </div>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.record-timeline {
  width: min(100%, 680px);
  padding: 18px 0 28px;
  border-top: 1px solid var(--studio-border);
}

.record-timeline__heading,
.record-timeline__entry-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.record-timeline__heading {
  justify-content: space-between;
  color: var(--studio-text-muted);
}

.record-timeline__title {
  margin: 0;
  color: var(--studio-text);
  font-size: 15px;
  font-weight: 700;
}

.record-timeline__hint,
.record-timeline__state,
.record-timeline__error {
  margin: 4px 0 0;
  color: var(--studio-text-muted);
  font-size: 12px;
}

.record-timeline__error { color: var(--studio-danger); }

.record-timeline__comment {
  display: grid;
  gap: 8px;
  margin: 18px 0;
}

.record-timeline__comment :deep(.field) { margin: 0; }

.record-timeline__entries {
  display: grid;
  gap: 0;
  margin: 20px 0 0;
  padding: 0;
  list-style: none;
}

.record-timeline__entry {
  display: grid;
  grid-template-columns: 12px 1fr;
  gap: 10px;
  min-height: 58px;
}

.record-timeline__marker {
  width: 8px;
  height: 8px;
  margin-top: 5px;
  border: 2px solid var(--studio-accent);
  border-radius: 50%;
}

.record-timeline__entry-body { padding-bottom: 16px; border-left: 1px solid var(--studio-border); padding-left: 12px; }
.record-timeline__entry-meta { flex-wrap: wrap; font-size: 12px; }
.record-timeline__entry-meta span { color: var(--studio-text-muted); }
.record-timeline__entry-meta time { margin-left: auto; color: var(--studio-text-muted); }
.record-timeline__message { margin: 7px 0 0; white-space: pre-wrap; font-size: 13px; }
.record-timeline__tags { display: flex; gap: 5px; margin-top: 7px; color: var(--studio-text-subtle); font-size: 10px; text-transform: uppercase; }
.record-timeline__tags span { padding: 2px 5px; border: 1px solid var(--studio-border); border-radius: 3px; }
</style>
